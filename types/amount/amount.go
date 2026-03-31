package amount

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const Precision = 1_000_000

var (
	zeroBigInt   = big.NewInt(0)
	precisionBig = big.NewInt(int64(Precision)) // 供 Mul/Div 等使用，避免重复分配
)

type Amount struct {
	val *big.Int
}

// Input 约束输入类型
type Input interface {
	~uint64 | ~int64 | ~float64 | ~float32 | ~string | ~int
}

// --- 构造函数 ---

// NewAmount 将数值转换为带精度的 Amount (e.g., 输入 1 -> 1,000,000)
func NewAmount[T Input](val T) Amount {
	f := toBigFloat(val)
	if f == nil {
		return Zero()
	}

	// 执行换算
	precision := big.NewFloat(float64(Precision))
	f.Mul(f, precision)

	res := new(big.Int)
	f.Int(res)
	return Amount{val: res}
}

// FromRaw 直接通过原始值构建 (e.g., 输入 1,000,000 -> 1.0)
func FromRaw[T Input](val T) Amount {
	f := toBigFloat(val)
	if f == nil {
		return Zero()
	}
	res := new(big.Int)
	f.Int(res)
	return Amount{val: res}
}

func Zero() Amount {
	return Amount{val: zeroBigInt} // 复用包级零值，避免每次分配
}

// 内部转换工具，减少代码冗余
func toBigFloat(val any) *big.Float {
	switch v := val.(type) {
	case int64:
		return new(big.Float).SetInt64(v)
	case uint64:
		return new(big.Float).SetUint64(v)
	case int:
		return new(big.Float).SetInt64(int64(v))
	case float64:
		return big.NewFloat(v)
	case float32:
		return big.NewFloat(float64(v))
	case string:
		f, ok := new(big.Float).SetString(v)
		if !ok {
			return nil
		}
		return f
	default:
		return nil
	}
}

// --- 基础方法 ---

func (a Amount) Int() *big.Int {
	if a.val == nil {
		return new(big.Int).Set(zeroBigInt)
	}
	return new(big.Int).Set(a.val) // 返回副本，防止外部修改
}

func (a Amount) IsZero() bool {
	return a.val == nil || a.val.Sign() == 0
}

func (a Amount) Sign() int {
	if a.val == nil {
		return 0
	}
	return a.val.Sign()
}

// --- 运算操作 (Immutable) ---

func (a Amount) Add(o Amount) Amount {
	res := new(big.Int).Add(a.safe(), o.safe())
	return Amount{val: res}
}

func (a Amount) Sub(o Amount) Amount {
	res := new(big.Int).Sub(a.safe(), o.safe())
	return Amount{val: res}
}

// Mul 乘法，结果保持精度 (raw 空间: a * o / Precision)
func (a Amount) Mul(o Amount) Amount {
	v, oVal := a.safe(), o.safe()
	tmp := new(big.Int).Mul(v, oVal)
	res := new(big.Int).Quo(tmp, precisionBig)
	return Amount{val: res}
}

// Div 除法，结果保持精度；除数为 0 时返回 Zero()
func (a Amount) Div(o Amount) Amount {
	if o.IsZero() {
		return Zero()
	}
	v, oVal := a.safe(), o.safe()
	tmp := new(big.Int).Mul(v, precisionBig)
	res := new(big.Int).Quo(tmp, oVal)
	return Amount{val: res}
}

// MulBy 乘以整数 k（同精度下等价于加 k 次自身）
func (a Amount) MulBy(k int64) Amount {
	if k == 0 {
		return Zero()
	}
	res := new(big.Int).Mul(a.safe(), big.NewInt(k))
	return Amount{val: res}
}

// DivBy 除以整数 k，向零取整；k == 0 时返回 Zero()
func (a Amount) DivBy(k int64) Amount {
	if k == 0 {
		return Zero()
	}
	res := new(big.Int).Quo(a.safe(), big.NewInt(k))
	return Amount{val: res}
}

func (a Amount) Neg() Amount {
	return Amount{val: new(big.Int).Neg(a.safe())}
}

// Abs 取绝对值
func (a Amount) Abs() Amount {
	if a.Sign() >= 0 {
		return a
	}
	return a.Neg()
}

// Cmp :  -1 if a < o, 0 if a == o, 1 if a > o
func (a Amount) Cmp(o Amount) int {
	return a.safe().Cmp(o.safe())
}

// Equals 是否相等
func (a Amount) Equals(o Amount) bool {
	return a.Cmp(o) == 0
}

// Less 是否小于
func (a Amount) Less(o Amount) bool {
	return a.Cmp(o) < 0
}

// Greater 是否大于
func (a Amount) Greater(o Amount) bool {
	return a.Cmp(o) > 0
}

// LessOrEqual 是否小于等于
func (a Amount) LessOrEqual(o Amount) bool {
	return a.Cmp(o) <= 0
}

// GreaterOrEqual 是否大于等于
func (a Amount) GreaterOrEqual(o Amount) bool {
	return a.Cmp(o) >= 0
}

// --- 转换与格式化 ---
func (a Amount) String() string {
	v := a.safe()
	if v.Sign() == 0 {
		return "0"
	}

	return v.String()
}

func (a Amount) Format() string {
	v := a.safe()
	if v.Sign() == 0 {
		return "0"
	}

	return formatFixed(v)
}

// Percent 格式化为百分比字符串，如 raw=1_000_000（表示 1.0）输出 "100"；负数输出如 "-50"、"-0.5"
func (a Amount) Percent() string {
	v := a.safe()
	if v.Sign() == 0 {
		return "0"
	}

	// 放大 100 倍计算百分比
	scaled := new(big.Int).Mul(v, big.NewInt(100))

	// 复用通用的格式化逻辑
	return formatFixed(scaled)
}

// Float64 将 Amount 转换为 float64 类型
// 注意：对于极大的数值，转换可能会导致精度丢失或溢出（返回 ±Inf）
func (a Amount) Float64() float64 {
	v := a.safe()
	if v.Sign() == 0 {
		return 0
	}

	// 使用 big.Rat 进行中转，base 是你的 precisionBig (10^6)
	// 这比手动做浮点除法更安全
	rat := new(big.Rat).SetFrac(v, big.NewInt(Precision))
	f, _ := rat.Float64()
	return f
}

// formatFixed 将 big.Int 按 precisionBig (10^6) 格式化为浮点字符串
func formatFixed(v *big.Int) string {
	if v.Sign() == 0 {
		return "0"
	}

	isNeg := v.Sign() < 0
	absV := new(big.Int).Abs(v)
	base := big.NewInt(Precision)

	// 分离整数和小数部分
	intPart := new(big.Int).Quo(absV, base)
	fracPart := new(big.Int).Mod(absV, base).Int64()

	var b strings.Builder
	b.Grow(16) // 预分配内存

	if isNeg {
		b.WriteByte('-')
	}
	b.WriteString(intPart.String())

	if fracPart > 0 {
		b.WriteByte('.')
		// 手动补齐 6 位零
		fStr := strconv.FormatInt(fracPart, 10)
		for i := 0; i < 6-len(fStr); i++ {
			b.WriteByte('0')
		}
		b.WriteString(fStr)
	}

	// 统一去除末尾多余的 0 和可能遗留的小数点
	res := strings.TrimRight(b.String(), "0")
	return strings.TrimRight(res, ".")
}

// --- 数据库/JSON 兼容 ---

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(`"` + a.safe().String() + `"`), nil
}

func (a *Amount) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		a.val = new(big.Int).Set(zeroBigInt)
		return nil
	}
	val, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("invalid amount value: %s", s)
	}
	a.val = val
	return nil
}

func (a Amount) Value() (driver.Value, error) {
	return a.safe().String(), nil
}

func (a *Amount) Scan(value any) error {
	var i *big.Int
	if err := scanBigInt(&i, value, "Amount"); err != nil {
		return err
	}
	a.val = i
	return nil
}

// safe 返回内部 big.Int，nil 时返回包级零值（只读使用），确保零值 Amount 可用
func (a Amount) safe() *big.Int {
	if a.val == nil {
		return zeroBigInt
	}
	return a.val
}

func scanBigInt(dst **big.Int, value any, name string) error {
	if value == nil {
		*dst = new(big.Int).Set(zeroBigInt)
		return nil
	}

	var s string

	switch v := value.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	case int64:
		*dst = big.NewInt(v)
		return nil
	case uint64:
		// 注意：big.NewInt 只接收 int64，所以 uint64 需要用 SetUint64
		*dst = new(big.Int).SetUint64(v)
		return nil
	case int:
		// 为了兼容性，也可以加上 int，强转为 int64 即可
		*dst = big.NewInt(int64(v))
		return nil
	default:
		return fmt.Errorf("unsupported Scan type for %s: %T", name, value)
	}

	i, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("invalid %s value: %s", name, s)
	}

	*dst = i

	return nil
}
