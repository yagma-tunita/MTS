// Package timeutil 提供中国时区（UTC+8）的时间工具函数。
//
// 为什么固定 UTC+8 而不是使用 time.Local 或 IANA 时区数据库：
//   - 航运系统涉及的所有业务操作（装货、到港、下单）都以中国时区为准。
//   - 如果使用 time.Local，当服务器部署在东八区以外的地区时，
//     时间会错乱。例如部署在新加坡的服务器（UTC+8，但时区名不同），
//     time.Local 可能返回不同的行为。
//   - 使用 time.FixedZone("CST", 8*3600) 硬编码了 +8 小时偏移，
//     不依赖操作系统时区配置，保证行为一致。
//
// 提供的功能：
//   - 当前时间：Now()
//   - 解析：ParseDate（日期）、ParseDateTime（日期+时间）
//   - 格式化：FormatDate（日期）、FormatDateTime（日期+时间）
//   - 工具：StartOfDay（当天起始）、Age（年龄计算）
//
// 使用示例：
//
//	now := timeutil.Now()             // 2026-07-06 14:30:00 +0800 CST
//	t, _ := timeutil.ParseDate("2026-07-06")
//	fmt.Println(timeutil.FormatDate(t)) // "2026-07-06"
//	age := timeutil.Age(birthDate)      // 周岁
package timeutil

import (
	"time"
)

// ChinaLocation 是东八区固定时区（CST，China Standard Time）。
//
// 使用 time.FixedZone 而非 time.LoadLocation("Asia/Shanghai")：
//   - FixedZone 不需要加载 IANA 时区数据库，不依赖操作系统。
//   - FixedZone 不支持夏令时（东八区也不存在夏令时）。
//   - 在 Docker 最小镜像（scratch/alpine）中也能正常工作。
var ChinaLocation = time.FixedZone("CST", 8*3600)

// Now 返回当前东八区时间。
//
// 等价于 time.Now().In(ChinaLocation)，保证返回的时间
// 的 Location 字段为 CST（东八区）。
func Now() time.Time {
	return time.Now().In(ChinaLocation)
}

// ParseDate 解析格式为 "2006-01-02" 的日期字符串，返回东八区时间。
//
// 返回的时间的时区为东八区，时分秒均为 0。
// 如果字符串格式不匹配，返回 error。
//
// 输入示例："2026-07-06" → 2026-07-06 00:00:00 +0800 CST
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, ChinaLocation)
}

// ParseDateTime 解析格式为 "2006-01-02 15:04:05" 的日期时间字符串，返回东八区时间。
//
// 输入示例："2026-07-06 14:30:00" → 2026-07-06 14:30:00 +0800 CST
func ParseDateTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, ChinaLocation)
}

// FormatDate 将时间格式化为 "2006-01-02"。
//
// 如果传入的 time.Time 携带其他时区，会自动转为东八区后再格式化。
func FormatDate(t time.Time) string {
	return t.In(ChinaLocation).Format("2006-01-02")
}

// FormatDateTime 将时间格式化为 "2006-01-02 15:04:05"。
//
// 如果传入的 time.Time 携带其他时区，会自动转为东八区后再格式化。
func FormatDateTime(t time.Time) string {
	return t.In(ChinaLocation).Format("2006-01-02 15:04:05")
}

// StartOfDay 返回指定日期当天的 00:00:00（东八区）。
//
// 输入任意时间，返回当天 0 点。用于日期范围查询的起始时间：
//
//	start := timeutil.StartOfDay(order.CreateTime)
//	db.Where("create_time >= ?", start)
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.In(ChinaLocation).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, ChinaLocation)
}

// Age 根据出生日期计算截止到今天的年龄（周岁）。
//
// 计算逻辑：
//  1. 计算年份差 (now.Year - birth.Year)。
//  2. 如果今年的生日还没过（now.YearDay < birth.YearDay），
//     减去 1 岁。
//
// 为什么用 YearDay 而不是比较月和日：
//   - YearDay 返回一年中的第几天（1-365/366），比分别比较
//     月和日更简洁。闰年的 2 月 29 日会被正确计算。
func Age(birthDate time.Time) int {
	now := Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}
