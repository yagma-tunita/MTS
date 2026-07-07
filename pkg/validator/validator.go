// Package validator 提供结构体参数校验功能。
//
// 基于 github.com/go-playground/validator/v10 —— Go 生态最流行的结构体验证库。
//
// 使用方式：在结构体字段上添加 validate tag，然后调用 validator.Validate(&req)。
//
//	type CreateReq struct {
//	    Name  string `json:"name" validate:"required,min=2"`
//	    Email string `json:"email" validate:"required,email"`
//	    Birth string `json:"birth" validate:"date"`           // 自定义规则
//	    Phone string `json:"phone" validate:"omitempty,phone"` // 自定义规则
//	}
//
// 自定义校验规则：
//   - date（日期格式）：验证是否为 "YYYY-MM-DD" 格式。
//   - phone（手机号）：验证是否为中国大陆手机号（1[3-9] 开头，11 位）。
//
// 空字符串处理：
//   - 自定义规则中，空字符串视为合法（通过 return true 不校验）。
//   - 如果要求必填，组合使用 "required,date" 或 "required,phone"。
//   - 这种方式允许字段在非必填时不校验格式，必填时同时校验存在和格式。
//
// 局限性：只做格式校验，不做业务存在性校验（如"用户名是否已存在"）。
// 业务存在性校验在 service 层完成。
package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// validate 是包级别的校验器实例。
//
// 为什么是包级单例而不是每次调用时新建：
//   - validator.New() 内部会缓存结构体的检查结果（通过反射），
//     多次调用浪费性能。
//   - 单例在 init() 中初始化，并注册了自定义规则，全局复用。
//
// 线程安全：go-playground/validator 的 Validate 方法是线程安全的，
// 多个 goroutine 可以同时调用。
var validate *validator.Validate

// init 注册自定义验证规则 date 和 phone
func init() {
	validate = validator.New()
	validate.RegisterValidation("date", validateDate)
	validate.RegisterValidation("phone", validatePhone)
}

// Validate 对结构体进行校验，返回校验错误。
//
// 参数 i 必须是结构体指针（或实现了 validation interface 的类型）。
// 校验规则通过 validate 结构体 tag 定义。
//
// 返回值（error）：
//   - nil: 校验通过。
//   - validator.ValidationErrors: 校验失败，包含每个失败字段的详细信息
//     （字段名、规则名、参数等）。可通过类型断言获取：
//
//     if err := validator.Validate(req); err != nil {
//         if verrs, ok := err.(validator.ValidationErrors); ok {
//             for _, ve := range verrs {
//                 fmt.Println(ve.Field(), ve.Tag())
//             }
//         }
//     }
func Validate(i interface{}) error {
	return validate.Struct(i)
}

// validateDate 自定义校验规则：校验字段是否为 "YYYY-MM-DD" 格式的日期字符串。
//
// 正则表达式：^\d{4}-\d{2}-\d{2}$
// 只校验格式，不校验日期是否合法（如 2026-02-30 会被误判为合法）。
// 如果需要校验日期的合法性，应使用 time.Parse 或 timeutil.ParseDate。
//
// 空字符串视为合法（return true），配合 "required" 使用来要求必填：
//   validate:"required,date"   — 必填且需符合日期格式
//   validate:"omitempty,date"  — 可选填，填了必须符合日期格式
func validateDate(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, fl.Field().String())
	return matched
}

// validatePhone 自定义校验规则：校验字段是否为中国大陆手机号。
//
// 正则表达式：^1[3-9]\d{9}$
//   - 1：中国手机号都以 1 开头
//   - [3-9]：第二位为 3-9（覆盖所有已发布的号段）
//   - \d{9}：后 9 位任意数字（总共 11 位）
//
// 空字符串视为合法（return true），配合 "required" 使用来要求必填。
func validatePhone(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, fl.Field().String())
	return matched
}

// RegisterCustomValidation 允许在运行时注册额外自定义校验规则。
//
// 使用场景：某个 handler 需要特殊的校验规则，不想全局注册在 init() 中。
// 注意：注册的 tag 不能与内置 tag 名称冲突。
//
// 示例：
//
//	validator.RegisterCustomValidation("password", func(fl validator.FieldLevel) bool {
//	    s := fl.Field().String()
//	    return len(s) >= 8 && strings.ContainsAny(s, "!@#$%")
//	})
func RegisterCustomValidation(tag string, fn validator.Func) {
	validate.RegisterValidation(tag, fn)
}

