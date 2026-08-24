// Package timeutil 统一使用 GMT+8（北京时间）处理业务时间戳。
package timeutil

import (
	"time"
)

const (
	LocationName = "Asia/Shanghai"
	Layout       = "2006-01-02 15:04:05"
)

var beijing = time.FixedZone("CST", 8*60*60)

// Beijing 返回固定 GMT+8 时区，避免容器缺 tzdata 时 LoadLocation 失败。
func Beijing() *time.Location {
	return beijing
}

// Now 返回当前北京时间（带时区）。入库使用 timestamptz，展示层格式化为 Layout。
func Now() time.Time {
	return time.Now().In(beijing)
}

// NowUTC 返回对应的 UTC 瞬间，供 pgx 写入 timestamptz。
func NowUTC() time.Time {
	return Now().UTC()
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(beijing).Format(Layout)
}

func Parse(s string) (time.Time, error) {
	return time.ParseInLocation(Layout, s, beijing)
}

// DurationMS 将 duration 转为毫秒整数，最小为 0。
func DurationMS(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms <= 0 {
		return 1
	}
	return ms
}
