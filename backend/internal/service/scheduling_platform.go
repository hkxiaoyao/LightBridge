package service

// 本文件集中封装 Gemini / Antigravity 合并后的“调度平台”语义。
//
// 背景：Antigravity 账号合并入 Gemini 平台后，在数据库中以
//   platform = "gemini" + sub_platform = "antigravity"
// 存储（见 Account.IsAntigravity）。但调度层仍以“平台别名”作为入参——
// 别名可能是 "anthropic" / "openai" / "gemini" / "antigravity"，分别来自
// 分组 platform、强制平台中间件（ForcePlatform）或快照 bucket。
//
// 由此带来两点必须统一处理的语义：
//   1. 查询：Antigravity 账号现位于 gemini 平台之下，任何可能涉及 Antigravity
//      的查询都需把 "antigravity" 别名翻译成实际 DB platform "gemini"。
//   2. 过滤：查询回来的 gemini 账号既含纯 Gemini 也含 Antigravity，需按目标
//      别名 + 是否混合调度进一步过滤。
//
// 调度语义总表（合并后）：
//   - platform=="antigravity"            → 仅 Antigravity 账号（专用/强制路由）。
//   - platform=="gemini"  且 useMixed    → 纯 Gemini + 启用 mixed 的 Antigravity。
//   - platform=="gemini"  且 !useMixed   → 仅纯 Gemini（排除 Antigravity）。
//   - platform=="anthropic" 且 useMixed  → 纯 Anthropic + 启用 mixed 的 Antigravity。
//   - platform=="anthropic" 且 !useMixed → 仅 Anthropic。
//   - 其他平台                            → account.Platform == platform。

// schedulingQueryPlatforms 返回为“服务于给定调度目标平台（别名）”需要从数据库
// 查询的 platform 列表。Antigravity 账号位于 gemini 平台下，故凡可能涉及
// Antigravity 的查询都需包含 "gemini"。
//
// 返回结果配合 accountServesSchedulingPlatform 做二次过滤使用。
func schedulingQueryPlatforms(platform string, useMixed bool) []string {
	switch platform {
	case PlatformAntigravity:
		// Antigravity 专用：账号在 gemini 平台下，查询 gemini 后按 sub_platform 过滤。
		return []string{PlatformGemini}
	case PlatformAnthropic:
		if useMixed {
			// Anthropic 混合：需要把（位于 gemini 平台下、启用 mixed 的）Antigravity 账号纳入。
			return []string{PlatformAnthropic, PlatformGemini}
		}
		return []string{PlatformAnthropic}
	case PlatformGemini:
		// 纯 Gemini 与 Antigravity 同处 gemini 平台，一次查询即可覆盖。
		return []string{PlatformGemini}
	default:
		return []string{platform}
	}
}

// accountServesSchedulingPlatform 判断账号是否可服务于给定调度目标平台（别名）。
//
// platform 为调度别名；useMixed 表示当前为混合调度（仅 anthropic / gemini 且非强制
// 平台时为 true）。这是合并后判断“某账号能否进入某平台调度集合”的唯一权威方式，
// 取代了历史上散落各处的 `account.Platform == X || (account.Platform == antigravity && mixed)`。
func accountServesSchedulingPlatform(a *Account, platform string, useMixed bool) bool {
	if a == nil {
		return false
	}
	// Antigravity 专用调度：只接受 Antigravity 账号，与 mixed 无关。
	if platform == PlatformAntigravity {
		return a.IsAntigravity()
	}
	if a.IsAntigravity() {
		// Antigravity 账号仅在混合调度（anthropic / gemini）且自身启用 mixed_scheduling 时被纳入。
		return useMixed &&
			(platform == PlatformAnthropic || platform == PlatformGemini) &&
			a.IsMixedSchedulingEnabled()
	}
	// 非 Antigravity 账号：平台精确匹配（纯 gemini 匹配 gemini，anthropic 匹配 anthropic）。
	return a.Platform == platform
}
