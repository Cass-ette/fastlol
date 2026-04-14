// i18n provides internationalization support for fastlol
package i18n

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Language codes
type Lang string

const (
	EN Lang = "en"
	ZH Lang = "zh"
	KO Lang = "ko"
)

var translations = map[Lang]map[string]string{
	EN: enStrings,
	ZH: zhStrings,
	KO: koStrings,
}

// enStrings - English (default)
var enStrings = map[string]string{
	// Root
	"root.short": "Fast League of Legends CLI tool",
	"root.long":  "Query champion counters, builds, runes, tier lists, and summoner profiles from the terminal.",

	// Common errors
	"error.no_riot_key":    "Riot API key not configured",
	"error.no_rapid_key":   "RapidAPI key not configured",
	"error.set_key_hint":   "  fastlol config set %s <your-key>",
	"error.fetch_failed":   "Failed to fetch data: %v",
	"error.not_found":      "Summoner not found: %v",
	"error.config_write":   "Failed to write config: %v",
	"error.marshal_config": "Failed to marshal config: %v",
	"error.unknown_key":    "Unknown config key: %s",
	"error.valid_keys":     "Valid keys: rapidapi_key, riot_api_key, default_region, language",

	// Common titles & labels
	"title.config":    "Configuration",
	"title.not_set":   "(not set)",
	"title.saved":     "Config saved to %s",
	"title.set":       "Set %s = %s",
	"tip.config_file": "  Config file: %s",

	// Flag descriptions
	"flag.region":        "Server region (kr, euw1, na1, etc.)",
	"flag.limit":         "Number of results to show",
	"flag.role":          "Role filter (top, jungle, mid, adc, support)",
	"flag.rapid_key":     "RapidAPI key (overrides config)",
	"flag.matches":       "Number of recent matches to show",
	"flag.mastery":       "Number of champion masteries to show",
	"flag.expand":        "Expand match N details (teammates/opponents)",
	"flag.mock":          "Use mock data (no API key needed)",

	// Commands - Tier
	"tier.use":         "tier",
	"tier.short":       "Show current patch tier list",
	"tier.long":        "View champion tier rankings for the current patch (requires RapidAPI key).\n\nExamples:\n  fastlol tier\n  fastlol tier --role mid\n  fastlol tier --role top -n 10",
	"tier.title":       "Tier List — Current Patch",
	"tier.no_data":     "No champions found for role: %s",
	"tier.headers":     "#,Champion,Tier,Role,Win Rate,Pick Rate,Ban Rate",

	// Commands - Top
	"top.use":             "top [tier]",
	"top.short":           "View Challenger/Grandmaster/Master leaderboard",
	"top.long":            "View top players in Challenger, Grandmaster, or Master tiers.\n\nExamples:\n  fastlol top challenger --region kr\n  fastlol top grandmaster --region kr\n  fastlol top master --region euw1\n  fastlol top -n 20",
	"top.tier.challenger": "Challenger",
	"top.tier.gm":         "Grandmaster",
	"top.tier.master":     "Master",
	"top.tier.unknown":    "Unknown tier: %s",
	"top.tier.valid":      "Valid: challenger / grandmaster / master",
	"top.title":           "Leaderboard | %s | %s",
	"top.no_data":         "No data available",
	"top.headers":         "Rank,Player,LP,W/L,Win Rate,Hot Streak",

	// Commands - Rank
	"rank.use":           "rank [summoner] [TAG]",
	"rank.short":         "Check summoner's ranked tier",
	"rank.long":          "Check a summoner's current ranked tier, LP, wins, losses, and win rate.\n\nExamples:\n  fastlol rank \"Bin\" KR1 --region kr\n  fastlol rank \"Caps\" EUW --region euw1",
	"rank.title":         "Rank: %s (%s)",
	"rank.no_data":       "No ranked data (unranked or privacy enabled)",
	"rank.privacy_warn":  "Ranked data unavailable due to privacy settings",
	"rank.stats":         "%dW %dL | %.1f%% WR",

	// Queue types
	"queue.solo":     "Ranked Solo",
	"queue.flex":     "Ranked Flex",
	"queue.tft":      "Teamfight Tactics",

	// Tiers (for display)
	"tier.iron":        "Iron",
	"tier.bronze":      "Bronze",
	"tier.silver":      "Silver",
	"tier.gold":        "Gold",
	"tier.platinum":    "Platinum",
	"tier.emerald":     "Emerald",
	"tier.diamond":     "Diamond",
	"tier.master":      "Master",
	"tier.gm":          "Grandmaster",
	"tier.challenger":  "Challenger",

	// Commands - Live
	"live.use":          "live [summoner] [TAG]",
	"live.short":        "Check if summoner is in an active game",
	"live.long":         "View real-time game data for an ongoing match.\n\nExamples:\n  fastlol live \"Bin\" KR1 --region kr\n  fastlol live \"Hide on bush\" KR",
	"live.title":        "Live Game: %s (%s)",
	"live.not_in_game":  "Not currently in a game",
	"live.tip":          "Can only view ongoing matches",
	"live.in_game":      "%s#%s is in a game!",
	"live.mode":         "Mode",
	"live.elapsed":      "Elapsed",
	"live.elapsed_fmt":  "%d:%02d",
	"live.team_blue":    "Blue Team",
	"live.team_red":     "Red Team",
	"live.headers":      "Champion,Player,Spells",
	"live.hidden":       "(hidden)",
	"live.spell_fmt":    "Spell%d",
	"live.bans_blue":    "Blue Bans",
	"live.bans_red":     "Red Bans",

	// Game modes
	"mode.classic":     "Classic",
	"mode.aram":        "ARAM",
	"mode.urf":         "URF",
	"mode.oneforall":   "One For All",
	"mode.nexusblitz":  "Nexus Blitz",
	"mode.cherry":      "Arena",

	// Summoner spells
	"spell.cleanse":    "Cleanse",
	"spell.exhaust":    "Exhaust",
	"spell.ignite":     "Ignite",
	"spell.flash":      "Flash",
	"spell.ghost":      "Ghost",
	"spell.heal":       "Heal",
	"spell.smite":      "Smite",
	"spell.teleport":   "Teleport",
	"spell.barrier":    "Barrier",
	"spell.snowball":   "Mark",

	// Commands - Rotation
	"rotation.use":         "rotation",
	"rotation.short":       "View free champion rotation",
	"rotation.long":        "View this week's free champion rotation.\n\nExamples:\n  fastlol rotation\n  fastlol rotation --region kr",
	"rotation.title":       "Free Rotation | %s",
	"rotation.header":      "Free Champions (%d)",
	"rotation.newbie":      "New Player (%d, level ≤ %d)",

	// Commands - Counter
	"counter.use":          "counter <champion>",
	"counter.short":        "View champion counter matchups",
	"counter.long":         "View counter picks and win rates for a champion (requires RapidAPI key).\n\nExamples:\n  fastlol counter Yone\n  fastlol counter Yasuo --role mid\n  fastlol counter \"Lee Sin\" --role jungle",
	"counter.title":        "Counter: %s",
	"counter.weak_against": "Weak Against (Hard Matchups)",
	"counter.strong_against": "Strong Against (Easy Matchups)",
	"counter.headers.basic": "Champion,Win Rate,Games",
	"counter.headers_weak":  "Champion,Their Win Rate",
	"counter.headers_strong": "Champion,Your Win Rate",
	"counter.stats":        "Stats",

	// Commands - Build
	"build.use":    "build <champion>",
	"build.short":  "View recommended build, runes, and win rate",
	"build.long":   "View recommended items, runes, and stats (requires RapidAPI key).\n\nExamples:\n  fastlol build Yone\n  fastlol build \"Lee Sin\"",
	"build.title":  "Build: %s",

	// Commands - Profile
	"profile.use":         "profile [summoner] [TAG]",
	"profile.short":       "View summoner profile (mastery + recent matches)",
	"profile.long":        "View a summoner's champion mastery and recent match history.\n\nExamples:\n  fastlol profile \"Bin\" KR1 --region kr\n  fastlol profile \"Caps\" EUW --region euw1 --matches 3",
	"profile.title.mock":  "[Mock] %s (%s)",
	"profile.title.real":  "Profile: %s (%s)",
	"profile.mastery":     "Champion Mastery",
	"profile.matches":     "Recent Matches",
	"profile.headers.mastery": "Rank,Champion,Level,Mastery,Last Played",
	"profile.headers.matches": ",Time,Mode,Champion,KDA,CS,Result,Duration",
	"profile.perfect":     "perfect",
	"profile.win":         "W",
	"profile.loss":        "L",
	"profile.canyon":      "Rift",
	"profile.aram":        "ARAM",
	"profile.expand_tip":  "Use --expand N to view match N details (teammates/opponents)",
	"profile.detail.title": "Match Details | %s | %s | %s",
	"profile.detail.headers": "Champion,Player,KDA,Result",
	"profile.team_blue":   "Blue (Win)",
	"profile.team_red":    "Red (Loss)",
	"profile.team_blue_loss": "Blue (Loss)",
	"profile.team_red_win":   "Red (Win)",
	"profile.devkey_tip":  "Note: Development API Key may not show summoner level/ranked data",

	// Commands - Challenges
	"challenges.use":         "challenges [summoner] [TAG]",
	"challenges.short":       "View challenge/achievement stats",
	"challenges.long":        "View a summoner's challenge and achievement stats.\n\nExamples:\n  fastlol challenges \"Bin\" KR1 --region kr\n  fastlol challenges -n 10",
	"challenges.title":       "Challenges: %s (%s)",
	"challenges.no_data":     "No challenge data",
	"challenges.privacy_warn": "Challenge data unavailable due to privacy settings",
	"challenges.headers":     "Rank,Level,Percentile,Value",

	// Challenge levels with emojis
	"level.challenger":   "👑 Challenger",
	"level.grandmaster":  "🥇 Grandmaster",
	"level.master":       "🥈 Master",
	"level.diamond":      "💎 Diamond",
	"level.emerald":      "💚 Emerald",
	"level.platinum":     "🟪 Platinum",
	"level.gold":         "🥉 Gold",
	"level.silver":       "🥈 Silver",
	"level.bronze":       "🥉 Bronze",
	"level.iron":         "⚫ Iron",

	// Commands - Clash
	"clash.use":          "clash [summoner] [TAG]",
	"clash.short":        "View Clash tournament records",
	"clash.long":         "View a summoner's Clash tournament participation.\n\nExamples:\n  fastlol clash \"Bin\" KR1 --region kr\n  fastlol clash \"Caps\" EUW --region euw1",
	"clash.title":        "Clash: %s (%s)",
	"clash.no_data":      "No Clash records (no active tournaments or not registered)",
	"clash.headers":      "Role,Summoner,Team ID",
	"clash.privacy_warn": "Clash data unavailable",
	"clash.reasons":      "Possible reasons:\n- No active Clash tournament\n- Not registered\n- Privacy settings",
	"clash.tip":          "Clash runs on weekends. Query with a recently active account.",

	// Positions
	"pos.top":      "Top",
	"pos.jungle":   "Jungle",
	"pos.mid":      "Mid",
	"pos.adc":      "ADC",
	"pos.support":  "Support",
	"pos.none":     "Not Set",
	"pos.pending":  "Pending",
	"pos.hidden":   "(hidden)",
	"pos.no_team":  "No Team",

	// Commands - Status
	"status.use":        "status [region]",
	"status.short":        "View server status",
	"status.long":         "Check server status and maintenance announcements.\n\nExamples:\n  fastlol status\n  fastlol status kr\n  fastlol status na1",
	"status.title":        "Server Status | %s",
	"status.server":       "Server: %s",
	"status.incidents":    "⚠️ Incidents/Announcements",
	"status.maintenance":  "🔧 Maintenance",
	"status.normal":       "✅ Server operating normally",
	"status.status.scheduled":  "Scheduled",
	"status.status.progress":   "In Progress",
	"status.status.resolved":   "Resolved",
	"status.status.critical":   "Critical",
	"status.region_code":   "Region Code: %s",

	// Region names (for status display)
	"region.kr":   "Korea",
	"region.euw1": "Europe West",
	"region.eun1": "Europe Nordic",
	"region.na1":  "North America",
	"region.jp1":  "Japan",
	"region.br1":  "Brazil",
	"region.la1":  "Latin America North",
	"region.la2":  "Latin America South",
	"region.oc1":  "Oceania",
	"region.tr1":  "Turkey",
	"region.ru":   "Russia",
	"region.sg2":  "Singapore",
	"region.ph2":  "Philippines",
	"region.th2":  "Thailand",
	"region.tw2":  "Taiwan",
	"region.vn2":  "Vietnam",
	"region.cn1":  "China",

	// Commands - Config
	"config.use":          "config",
	"config.short":        "Manage configuration",
	"config.set.use":      "set <key> <value>",
	"config.set.short":      "Set a config value",
	"config.set.long":       "Set a config value. Keys: rapidapi_key, riot_api_key, default_region, language",
	"config.show.use":       "show",
	"config.show.short":     "Show current configuration",

	// Language setting
	"lang.current":  "Current language: %s",
	"lang.changed":  "Language changed to: %s",
	"lang.invalid":  "Invalid language: %s. Valid: en, zh, ko",
	"lang.en":       "English",
	"lang.zh":       "中文",
	"lang.ko":       "한국어",
}

// zhStrings - Chinese
var zhStrings = map[string]string{
	"root.short": "英雄联盟数据查询 CLI 工具",
	"root.long":  "在终端查询英雄克制、出装、符文、强度排名、召唤师战绩、排位段位、观战等数据。",

	"error.no_riot_key":    "未配置 Riot API key",
	"error.no_rapid_key":   "未配置 RapidAPI key",
	"error.set_key_hint":   "  fastlol config set %s <你的key>",
	"error.fetch_failed":   "获取数据失败: %v",
	"error.not_found":      "未找到该召唤师: %v",
	"error.config_write":   "配置写入失败: %v",
	"error.marshal_config": "配置序列化失败: %v",
	"error.unknown_key":    "未知配置项: %s",
	"error.valid_keys":     "有效配置项: rapidapi_key, riot_api_key, default_region, language",

	"title.config":    "配置",
	"title.not_set":   "(未设置)",
	"title.saved":     "配置已保存到 %s",
	"title.set":       "设置 %s = %s",
	"tip.config_file": "  配置文件: %s",

	"flag.region":        "服务器区域 (kr, euw1, na1, 等)",
	"flag.limit":         "显示数量",
	"flag.role":          "位置筛选 (top, jungle, mid, adc, support)",
	"flag.rapid_key":     "RapidAPI key (覆盖配置文件)",
	"flag.matches":       "显示近期比赛数量",
	"flag.mastery":       "显示英雄成就数量",
	"flag.expand":        "展开第 N 场比赛详情（队友/对手）",
	"flag.mock":          "使用模拟数据（无需 API key）",

	"tier.use":         "tier",
	"tier.short":       "查看当前版本强势英雄",
	"tier.long":        "查看当前版本英雄强度排名（需要 RapidAPI key）。\n\n用法示例:\n  fastlol tier\n  fastlol tier --role mid\n  fastlol tier --role top -n 10",
	"tier.title":       "📊 英雄强度排名 — 当前版本",
	"tier.no_data":     "该位置没有数据: %s",
	"tier.headers":     "#,英雄,Tier,位置,胜率,选取率,Ban率",

	"top.use":             "top [段位]",
	"top.short":           "查看王者/宗师/大师榜单",
	"top.long":            "查看王者、宗师、大师段位排行榜。\n\n用法示例:\n  fastlol top challenger --region kr\n  fastlol top grandmaster --region kr\n  fastlol top master --region euw1\n  fastlol top -n 20",
	"top.tier.challenger": "王者",
	"top.tier.gm":         "宗师",
	"top.tier.master":     "大师",
	"top.tier.unknown":    "未知段位: %s",
	"top.tier.valid":      "可选: challenger / grandmaster / master",
	"top.title":           "🏆 %s 榜单 | %s",
	"top.no_data":         "暂无数据",
	"top.headers":         "排名,玩家,LP,胜负,胜率,状态",

	"rank.use":           "rank [选手名] [TAG]",
	"rank.short":         "查询排位段位",
	"rank.long":          "查询召唤师当前排位段位、LP、胜败场和胜率。\n\n用法示例:\n  fastlol rank \"Bin\" KR1 --region kr\n  fastlol rank \"Caps\" EUW --region euw1",
	"rank.title":         "🔍 排位查询: %s (%s)",
	"rank.no_data":       "暂无排位数据（未打定级赛或设置了隐私）",
	"rank.privacy_warn":  "排位数据可能因隐私设置无法获取",
	"rank.stats":         "%d胜 %d败 | 胜率 %.1f%%",

	"queue.solo":     "单排/双排",
	"queue.flex":     "灵活排位",
	"queue.tft":      "云顶之弈",

	"tier.iron":        "坚韧黑铁",
	"tier.bronze":      "英勇青铜",
	"tier.silver":      "不屈白银",
	"tier.gold":        "荣耀黄金",
	"tier.platinum":    "华贵铂金",
	"tier.emerald":     "流光翡翠",
	"tier.diamond":     "璀璨钻石",
	"tier.master":      "超凡大师",
	"tier.gm":          "傲世宗师",
	"tier.challenger":  "最强王者",

	"live.use":          "live [选手名] [TAG]",
	"live.short":          "查看是否正在对局",
	"live.long":           "查看召唤师是否正在对局中。\n\n用法示例:\n  fastlol live \"Bin\" KR1 --region kr\n  fastlol live \"허거덩\" 0303 --region kr",
	"live.title":          "👁️ 实时对局: %s (%s)",
	"live.not_in_game":    "当前不在对局中",
	"live.tip":            "💡 只能查看正在进行的对局",
	"live.in_game":        "%s#%s 正在对局中！",
	"live.mode":           "模式",
	"live.elapsed":        "已进行",
	"live.elapsed_fmt":    "%d:%02d",
	"live.team_blue":      "🔵 蓝方",
	"live.team_red":       "🔴 红方",
	"live.headers":        "英雄,玩家,召唤师技能",
	"live.hidden":         "(隐藏)",
	"live.spell_fmt":      "技能%d",
	"live.bans_blue":      "🚫 蓝方Ban",
	"live.bans_red":       "🚫 红方Ban",

	"mode.classic":     "经典模式",
	"mode.aram":        "极地大乱斗",
	"mode.urf":         "无限火力",
	"mode.oneforall":   "克隆大作战",
	"mode.nexusblitz":  "极限闪击",
	"mode.cherry":      "斗魂竞技场",

	"spell.cleanse":    "净化",
	"spell.exhaust":    "虚弱",
	"spell.ignite":     "引燃",
	"spell.flash":      "闪现",
	"spell.ghost":      "幽灵疾步",
	"spell.heal":       "治疗",
	"spell.smite":      "惩戒",
	"spell.teleport":   "传送",
	"spell.barrier":    "屏障",
	"spell.snowball":   "雪球",

	"rotation.use":         "rotation",
	"rotation.short":       "查看周免英雄",
	"rotation.long":        "查看本周免费英雄列表。\n\n用法示例:\n  fastlol rotation\n  fastlol rotation --region kr",
	"rotation.title":       "🔄 周免英雄 | %s",
	"rotation.header":      "本周免费英雄 (%d)",
	"rotation.newbie":      "新手专属周免 (%d个，等级≤%d)",

	"counter.use":          "counter <英雄名>",
	"counter.short":          "查询英雄克制关系",
	"counter.long":           "查询英雄克制关系和胜率（需要 RapidAPI key）。\n\n用法示例:\n  fastlol counter Yone\n  fastlol counter Yasuo --role mid\n  fastlol counter \"Lee Sin\" --role jungle",
	"counter.title":          "⚔️ 克制查询: %s",
	"counter.weak_against":   "被克制 (难打的对手)",
	"counter.strong_against": "克制 (好打的对手)",
	"counter.headers.basic":  "英雄,胜率,对局数",
	"counter.headers_weak":   "英雄,对方胜率",
	"counter.headers_strong": "英雄,你的胜率",
	"counter.stats":          "基础数据",

	"build.use":    "build <英雄名>",
	"build.short":  "查询推荐出装/符文",
	"build.long":   "查询推荐出装、符文和胜率（需要 RapidAPI key）。\n\n用法示例:\n  fastlol build Yone\n  fastlol build \"Lee Sin\"",
	"build.title":  "🛡️ 出装查询: %s",

	"profile.use":         "profile [选手名] [TAG]",
	"profile.short":       "查询召唤师战绩",
	"profile.long":        "查询召唤师英雄成就和近期比赛。\n\n用法示例:\n  fastlol profile \"Bin\" KR1 --region kr\n  fastlol profile \"Caps\" EUW --region euw1 --matches 3",
	"profile.title.mock":  "[模拟] %s (%s)",
	"profile.title.real":  "🔍 查询: %s (%s)",
	"profile.mastery":     "🎮 英雄成就",
	"profile.matches":     "📊 近期比赛",
	"profile.headers.mastery": "排名,英雄,等级,熟练度,最近使用",
	"profile.headers.matches": ",时间,模式,英雄,KDA,补刀,结果,时长",
	"profile.perfect":     "完美",
	"profile.win":         "胜",
	"profile.loss":        "败",
	"profile.canyon":      "峡谷",
	"profile.aram":        "乱斗",
	"profile.expand_tip":  "💡 使用 --expand N 查看第 N 场详情",
	"profile.detail.title": "📋 比赛详情 | %s | %s | %s",
	"profile.detail.headers": "英雄,玩家,KDA,结果",
	"profile.team_blue":   "蓝方",
	"profile.team_red":    "红方",
	"profile.team_blue_loss": "蓝方 (败)",
	"profile.team_red_win":   "红方 (胜)",
	"profile.devkey_tip":  "💡 Development API Key 限制: 部分数据可能无法获取",

	"challenges.use":         "challenges [选手名] [TAG]",
	"challenges.short":       "查询成就挑战",
	"challenges.long":        "查询召唤师成就挑战数据。\n\n用法示例:\n  fastlol challenges \"Bin\" KR1 --region kr\n  fastlol challenges -n 10",
	"challenges.title":       "🏅 成就查询: %s (%s)",
	"challenges.no_data":     "暂无成就数据",
	"challenges.privacy_warn": "成就数据可能因隐私设置无法获取",
	"challenges.headers":     "排名,成就等级,百分位,当前值",

	"level.challenger":   "👑 最强王者",
	"level.grandmaster":  "🥇 傲世宗师",
	"level.master":       "🥈 超凡大师",
	"level.diamond":      "💎 璀璨钻石",
	"level.emerald":      "💚 流光翡翠",
	"level.platinum":     "🟪 华贵铂金",
	"level.gold":         "🥉 荣耀黄金",
	"level.silver":       "🥛 不屈白银",
	"level.bronze":       "🟤 英勇青铜",
	"level.iron":         "⚫ 坚韧黑铁",

	"clash.use":          "clash [选手名] [TAG]",
	"clash.short":        "查询战队赛记录",
	"clash.long":         "查询召唤师战队赛参赛记录。\n\n用法示例:\n  fastlol clash \"Bin\" KR1 --region kr\n  fastlol clash \"Caps\" EUW --region euw1",
	"clash.title":        "⚔️ 战队赛查询: %s (%s)",
	"clash.no_data":      "暂无战队赛记录",
	"clash.headers":      "位置,召唤师名,队伍ID",
	"clash.privacy_warn": "战队赛数据无法获取",
	"clash.reasons":      "可能原因:\n- 当前无进行中战队赛\n- 未报名\n- 隐私设置",
	"clash.tip":          "💡 战队赛每周末举行，请使用最近有赛事的账号查询",

	"pos.top":      "上路",
	"pos.jungle":   "打野",
	"pos.mid":      "中路",
	"pos.adc":      "下路",
	"pos.support":  "辅助",
	"pos.none":     "未指定",
	"pos.pending":  "待分配",
	"pos.hidden":   "(隐藏)",
	"pos.no_team":  "无队伍",

	"status.use":        "status [区域]",
	"status.short":        "查看服务器状态",
	"status.long":         "查看服务器状态和维护公告。\n\n用法示例:\n  fastlol status\n  fastlol status kr\n  fastlol status na1",
	"status.title":        "🛠️ 服务器状态 | %s",
	"status.server":       "服务器: %s",
	"status.incidents":    "⚠️ 有事件/公告",
	"status.maintenance":  "🔧 维护公告",
	"status.normal":       "✅ 服务器正常运行，无公告",
	"status.status.scheduled":  "已计划",
	"status.status.progress":   "进行中",
	"status.status.resolved":   "已解决",
	"status.status.critical":   "严重",
	"status.region_code":   "区域代码: %s",

	"region.kr":   "韩国",
	"region.euw1": "西欧",
	"region.eun1": "东欧",
	"region.na1":  "北美",
	"region.jp1":  "日本",
	"region.br1":  "巴西",
	"region.la1":  "拉美北",
	"region.la2":  "拉美南",
	"region.oc1":  "大洋洲",
	"region.tr1":  "土耳其",
	"region.ru":   "俄罗斯",
	"region.sg2":  "新加坡",
	"region.ph2":  "菲律宾",
	"region.th2":  "泰国",
	"region.tw2":  "台湾",
	"region.vn2":  "越南",
	"region.cn1":  "中国",

	"config.use":          "config",
	"config.short":        "管理配置",
	"config.set.use":      "set <key> <value>",
	"config.set.short":      "设置配置项",
	"config.set.long":       "设置配置项。可选: rapidapi_key, riot_api_key, default_region, language",
	"config.show.use":       "show",
	"config.show.short":     "显示当前配置",

	"lang.current":  "当前语言: %s",
	"lang.changed":  "语言已切换为: %s",
	"lang.invalid":  "无效语言: %s。可选: en, zh, ko",
	"lang.en":       "English",
	"lang.zh":       "中文",
	"lang.ko":       "한국어",
}

// koStrings - Korean
var koStrings = map[string]string{
	"root.short": "리그 오브 레전드 CLI 도구",
	"root.long":  "터미널에서 챔피언 카운터, 빌드, 룬, 티어 리스트, 소환사 프로필을 조회하세요.",

	"error.no_riot_key":    "Riot API 키가 설정되지 않았습니다",
	"error.no_rapid_key":   "RapidAPI 키가 설정되지 않았습니다",
	"error.set_key_hint":   "  fastlol config set %s <키>",
	"error.fetch_failed":   "데이터 가져오기 실패: %v",
	"error.not_found":      "소환사를 찾을 수 없습니다: %v",
	"error.config_write":   "설정 저장 실패: %v",
	"error.marshal_config": "설정 직렬화 실패: %v",
	"error.unknown_key":    "알 수 없는 설정: %s",
	"error.valid_keys":     "유효한 설정: rapidapi_key, riot_api_key, default_region, language",

	"title.config":    "설정",
	"title.not_set":   "(설정 안 됨)",
	"title.saved":     "설정이 %s에 저장됨",
	"title.set":       "%s = %s 설정",
	"tip.config_file": "  설정 파일: %s",

	"flag.region":        "서버 지역 (kr, euw1, na1 등)",
	"flag.limit":         "표시할 결과 수",
	"flag.role":          "역할 필터 (top, jungle, mid, adc, support)",
	"flag.rapid_key":     "RapidAPI 키 (설정 덮어쓰기)",
	"flag.matches":       "표시할 최근 전적 수",
	"flag.mastery":       "표시할 챔피언 숙련도 수",
	"flag.expand":        "N번째 경기 상세 보기 (팀원/상대)",
	"flag.mock":          "모의 데이터 사용 (API 키 불필요)",

	"tier.use":         "tier",
	"tier.short":       "현재 패치 티어 리스트 보기",
	"tier.long":        "현재 패치 챔피언 티어 순위 보기 (RapidAPI 필요).\n\n예시:\n  fastlol tier\n  fastlol tier --role mid\n  fastlol tier --role top -n 10",
	"tier.title":       "📊 챔피언 티어 — 현재 패치",
	"tier.no_data":     "해당 역할에 데이터 없음: %s",
	"tier.headers":     "#,챔피언,티어,역할,승률,픽률,밴률",

	"top.use":             "top [티어]",
	"top.short":           "챌린저/그랜드마스터/마스터 순위 보기",
	"top.long":            "챌린저, 그랜드마스터, 마스터 티어 상위 플레이어 보기.\n\n예시:\n  fastlol top challenger --region kr\n  fastlol top grandmaster --region kr\n  fastlol top master --region euw1\n  fastlol top -n 20",
	"top.tier.challenger": "챌린저",
	"top.tier.gm":         "그랜드마스터",
	"top.tier.master":     "마스터",
	"top.tier.unknown":    "알 수 없는 티어: %s",
	"top.tier.valid":      "유효: challenger / grandmaster / master",
	"top.title":           "🏆 %s 순위 | %s",
	"top.no_data":         "데이터 없음",
	"top.headers":         "순위,플레이어,LP,승/패,승률,연승",

	"rank.use":           "rank [소환사] [TAG]",
	"rank.short":         "랭크 티어 조회",
	"rank.long":          "소환사의 현재 랭크 티어, LP, 승패 및 승률 조회.\n\n예시:\n  fastlol rank \"Bin\" KR1 --region kr\n  fastlol rank \"Caps\" EUW --region euw1",
	"rank.title":         "🔍 랭크 조회: %s (%s)",
	"rank.no_data":       "랭크 데이터 없음 (언랭크 또는 비공개)",
	"rank.privacy_warn":  "비공개 설정으로 인해 랭크 데이터를 가져올 수 없습니다",
	"rank.stats":         "%d승 %d패 | 승률 %.1f%%",

	"queue.solo":     "솔로랭크",
	"queue.flex":     "자유랭크",
	"queue.tft":      "전략적 팀 전투",

	"tier.iron":        "아이언",
	"tier.bronze":      "브론즈",
	"tier.silver":      "실버",
	"tier.gold":        "골드",
	"tier.platinum":    "플래티넘",
	"tier.emerald":     "에메랄드",
	"tier.diamond":     "다이아몬드",
	"tier.master":      "마스터",
	"tier.gm":          "그랜드마스터",
	"tier.challenger":  "챌린저",

	"live.use":          "live [소환사] [TAG]",
	"live.short":        "실시간 게임 확인",
	"live.long":         "소환사가 현재 게임 중인지 확인합니다.\n\n예시:\n  fastlol live \"Bin\" KR1 --region kr\n  fastlol live \"Hide on bush\" KR",
	"live.title":        "👁️ 실시간 게임: %s (%s)",
	"live.not_in_game":  "현재 게임 중이 아님",
	"live.tip":          "💡 진행 중인 게임만 볼 수 있습니다",
	"live.in_game":      "%s#%s 님이 게임 중입니다!",
	"live.mode":         "모드",
	"live.elapsed":      "경과 시간",
	"live.elapsed_fmt":  "%d:%02d",
	"live.team_blue":    "🔵 블루팀",
	"live.team_red":     "🔴 레드팀",
	"live.headers":      "챔피언,플레이어,스펠",
	"live.hidden":       "(숨김)",
	"live.spell_fmt":    "스펠%d",
	"live.bans_blue":    "🚫 블루 밴",
	"live.bans_red":     "🚫 레드 밴",

	"mode.classic":     "클래식",
	"mode.aram":        "칼바람 나락",
	"mode.urf":         "URF",
	"mode.oneforall":   "단일 챔피언",
	"mode.nexusblitz":  "넥서스 블리츠",
	"mode.cherry":      "투기장",

	"spell.cleanse":    "정화",
	"spell.exhaust":    "탈진",
	"spell.ignite":     "점화",
	"spell.flash":      "점멸",
	"spell.ghost":      "유체화",
	"spell.heal":       "회복",
	"spell.smite":      "강타",
	"spell.teleport":   "순간이동",
	"spell.barrier":    "방어막",
	"spell.snowball":   "표식",

	"rotation.use":         "rotation",
	"rotation.short":       "무료 챔피언 로테이션",
	"rotation.long":        "이번 주 무료 챔피언 목록 보기.\n\n예시:\n  fastlol rotation\n  fastlol rotation --region kr",
	"rotation.title":       "🔄 무료 로테이션 | %s",
	"rotation.header":      "무료 챔피언 (%d)",
	"rotation.newbie":      "신규 플레이어 (%d개, 레벨 ≤ %d)",

	"counter.use":          "counter <챔피언>",
	"counter.short":          "챔피언 카운터 조회",
	"counter.long":           "챔피언 카운터 관계 및 승률 조회 (RapidAPI 필요).\n\n예시:\n  fastlol counter Yone\n  fastlol counter Yasuo --role mid\n  fastlol counter \"Lee Sin\" --role jungle",
	"counter.title":          "⚔️ 카운터: %s",
	"counter.weak_against":   "약함 (카운터당함)",
	"counter.strong_against": "강함 (카운터함)",
	"counter.headers.basic":  "챔피언,승률,게임 수",
	"counter.headers_weak":   "챔피언,상대 승률",
	"counter.headers_strong": "챔피언,내 승률",
	"counter.stats":          "기본 데이터",

	"build.use":    "build <챔피언>",
	"build.short":  "추천 빌드/룬 조회",
	"build.long":   "추천 아이템, 룬 및 통계 조회 (RapidAPI 필요).\n\n예시:\n  fastlol build Yone\n  fastlol build \"Lee Sin\"",
	"build.title":  "🛡️ 빌드: %s",

	"profile.use":         "profile [소환사] [TAG]",
	"profile.short":       "소환사 프로필 조회",
	"profile.long":        "소환사 챔피언 숙련도 및 최근 전적 조회.\n\n예시:\n  fastlol profile \"Bin\" KR1 --region kr\n  fastlol profile \"Caps\" EUW --region euw1 --matches 3",
	"profile.title.mock":  "[모의] %s (%s)",
	"profile.title.real":  "🔍 조회: %s (%s)",
	"profile.mastery":     "🎮 챔피언 숙련도",
	"profile.matches":     "📊 최근 전적",
	"profile.headers.mastery": "순위,챔피언,레벨,숙련도,최근 플레이",
	"profile.headers.matches": ",시간,모드,챔피언,KDA,CS,결과,시간",
	"profile.perfect":     "완벽",
	"profile.win":         "승",
	"profile.loss":        "패",
	"profile.canyon":      "소환사의 협곡",
	"profile.aram":        "칼바람",
	"profile.expand_tip":  "💡 --expand N으로 N번째 경기 상세 보기",
	"profile.detail.title": "📋 경기 상세 | %s | %s | %s",
	"profile.detail.headers": "챔피언,플레이어,KDA,결과",
	"profile.team_blue":   "블루팀",
	"profile.team_red":    "레드팀",
	"profile.team_blue_loss": "블루팀 (패)",
	"profile.team_red_win":   "레드팀 (승)",
	"profile.devkey_tip":  "💡 개발 API 키 제한: 일부 데이터가 표시되지 않을 수 있음",

	"challenges.use":         "challenges [소환사] [TAG]",
	"challenges.short":       "도전 과제 조회",
	"challenges.long":        "소환사 도전 과제 및 업적 통계 조회.\n\n예시:\n  fastlol challenges \"Bin\" KR1 --region kr\n  fastlol challenges -n 10",
	"challenges.title":       "🏅 도전 과제: %s (%s)",
	"challenges.no_data":     "도전 과제 데이터 없음",
	"challenges.privacy_warn": "비공개 설정으로 인해 도전 과제 데이터를 가져올 수 없습니다",
	"challenges.headers":     "순위,레벨,백분위,값",

	"level.challenger":   "👑 챌린저",
	"level.grandmaster":  "🥇 그랜드마스터",
	"level.master":       "🥈 마스터",
	"level.diamond":      "💎 다이아몬드",
	"level.emerald":      "💚 에메랄드",
	"level.platinum":     "🟪 플래티넘",
	"level.gold":         "🥉 골드",
	"level.silver":       "🥈 실버",
	"level.bronze":       "🥉 브론즈",
	"level.iron":         "⚫ 아이언",

	"clash.use":          "clash [소환사] [TAG]",
	"clash.short":        "클래시 대회 기록",
	"clash.long":         "소환사 클래시 대회 참가 기록 조회.\n\n예시:\n  fastlol clash \"Bin\" KR1 --region kr\n  fastlol clash \"Caps\" EUW --region euw1",
	"clash.title":        "⚔️ 클래시: %s (%s)",
	"clash.no_data":      "클래시 기록 없음",
	"clash.headers":      "역할,소환사,팀 ID",
	"clash.privacy_warn": "클래시 데이터를 가져올 수 없음",
	"clash.reasons":      "가능한 이유:\n- 진행 중인 클래시 없음\n- 미등록\n- 비공개 설정",
	"clash.tip":          "💡 클래시는 주말에 진행됩니다. 최근 활동 계정으로 조회하세요.",

	"pos.top":      "탑",
	"pos.jungle":   "정글",
	"pos.mid":      "미드",
	"pos.adc":      "원딜",
	"pos.support":  "서폿",
	"pos.none":     "미지정",
	"pos.pending":  "대기 중",
	"pos.hidden":   "(숨김)",
	"pos.no_team":  "무소속",

	"status.use":        "status [지역]",
	"status.short":        "서버 상태 확인",
	"status.long":         "서버 상태 및 유지보수 공지 확인.\n\n예시:\n  fastlol status\n  fastlol status kr\n  fastlol status na1",
	"status.title":        "🛠️ 서버 상태 | %s",
	"status.server":       "서버: %s",
	"status.incidents":    "⚠️ 사건/공지",
	"status.maintenance":  "🔧 유지보수",
	"status.normal":       "✅ 서버 정상 작동 중",
	"status.status.scheduled":  "예정됨",
	"status.status.progress":   "진행 중",
	"status.status.resolved":   "해결됨",
	"status.status.critical":   "심각",
	"status.region_code":   "지역 코드: %s",

	"region.kr":   "한국",
	"region.euw1": "서유럽",
	"region.eun1": "북동유럽",
	"region.na1":  "북미",
	"region.jp1":  "일본",
	"region.br1":  "브라질",
	"region.la1":  "라틴 북부",
	"region.la2":  "라틴 남부",
	"region.oc1":  "오세아니아",
	"region.tr1":  "터키",
	"region.ru":   "러시아",
	"region.sg2":  "싱가포르",
	"region.ph2":  "필리핀",
	"region.th2":  "태국",
	"region.tw2":  "대만",
	"region.vn2":  "베트남",
	"region.cn1":  "중국",

	"config.use":          "config",
	"config.short":        "설정 관리",
	"config.set.use":      "set <key> <value>",
	"config.set.short":      "설정 값 변경",
	"config.set.long":       "설정 값 변경. 옵션: rapidapi_key, riot_api_key, default_region, language",
	"config.show.use":       "show",
	"config.show.short":     "현재 설정 보기",

	"lang.current":  "현재 언어: %s",
	"lang.changed":  "언어 변경됨: %s",
	"lang.invalid":  "유효하지 않은 언어: %s. 옵션: en, zh, ko",
	"lang.en":       "English",
	"lang.zh":       "中文",
	"lang.ko":       "한국어",
}

// GetLanguage returns the current language from config, defaulting to EN
func GetLanguage() Lang {
	lang := viper.GetString("language")
	switch lang {
	case "zh", "zh-CN", "zh-TW", "zh-HK":
		return ZH
	case "ko", "ko-KR":
		return KO
	case "en", "en-US", "en-GB":
		return EN
	default:
		// Check environment variable
		if envLang := os.Getenv("FASTLOL_LANG"); envLang != "" {
			switch envLang {
			case "zh", "zh-CN", "zh-TW":
				return ZH
			case "ko", "ko-KR":
				return KO
			}
		}
		return EN
	}
}

// T returns the translation for a given key
func T(key string) string {
	return Tl(GetLanguage(), key)
}

// Tl returns the translation for a given language and key
func Tl(lang Lang, key string) string {
	if dict, ok := translations[lang]; ok {
		if str, ok := dict[key]; ok {
			return str
		}
	}
	// Fallback to English
	if str, ok := enStrings[key]; ok {
		return str
	}
	return key
}

// Tf returns a formatted translation
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// GetLocalizedTier returns localized tier name
func GetLocalizedTier(tier string) string {
	key := "tier." + tierToKey(tier)
	return T(key)
}

// GetLocalizedLevel returns localized challenge level with emoji
func GetLocalizedLevel(level string) string {
	key := "level." + levelToKey(level)
	return T(key)
}

// GetLocalizedRole returns localized position name
func GetLocalizedRole(pos string) string {
	key := "pos." + positionToKey(pos)
	return T(key)
}

// GetLocalizedQueue returns localized queue type
func GetLocalizedQueue(queue string) string {
	switch queue {
	case "RANKED_SOLO_5x5":
		return T("queue.solo")
	case "RANKED_FLEX_SR":
		return T("queue.flex")
	case "RANKED_TFT":
		return T("queue.tft")
	default:
		return queue
	}
}

// GetLocalizedRegion returns localized region name
func GetLocalizedRegion(region string) string {
	key := "region." + region
	return T(key)
}

// GetLocalizedMode returns localized game mode
func GetLocalizedMode(mode string) string {
	switch mode {
	case "CLASSIC":
		return T("mode.classic")
	case "ARAM":
		return T("mode.aram")
	case "URF":
		return T("mode.urf")
	case "ONEFORALL":
		return T("mode.oneforall")
	case "NEXUSBLITZ":
		return T("mode.nexusblitz")
	case "CHERRY":
		return T("mode.cherry")
	default:
		return mode
	}
}

// GetLocalizedSpell returns localized summoner spell name
func GetLocalizedSpell(spellID int64) string {
	switch spellID {
	case 1:
		return T("spell.cleanse")
	case 2:
		return T("spell.exhaust")
	case 3, 14:
		return T("spell.ignite")
	case 4:
		return T("spell.flash")
	case 6:
		return T("spell.ghost")
	case 7:
		return T("spell.heal")
	case 11:
		return T("spell.smite")
	case 12:
		return T("spell.teleport")
	case 21:
		return T("spell.barrier")
	case 32:
		return T("spell.snowball")
	default:
		return fmt.Sprintf(T("live.spell_fmt"), spellID)
	}
}

// Helper functions
func tierToKey(tier string) string {
	switch tier {
	case "IRON", "Iron", "iron":
		return "iron"
	case "BRONZE", "Bronze", "bronze":
		return "bronze"
	case "SILVER", "Silver", "silver":
		return "silver"
	case "GOLD", "Gold", "gold":
		return "gold"
	case "PLATINUM", "Platinum", "platinum":
		return "platinum"
	case "EMERALD", "Emerald", "emerald":
		return "emerald"
	case "DIAMOND", "Diamond", "diamond":
		return "diamond"
	case "MASTER", "Master", "master":
		return "master"
	case "GRANDMASTER", "Grandmaster", "grandmaster":
		return "gm"
	case "CHALLENGER", "Challenger", "challenger":
		return "challenger"
	default:
		return tierToLower(tier)
	}
}

func positionToKey(pos string) string {
	switch pos {
	case "TOP":
		return "top"
	case "JGL", "JUNGLE":
		return "jungle"
	case "MID":
		return "mid"
	case "ADC":
		return "adc"
	case "SUP", "SUPPORT":
		return "support"
	case "NONE":
		return "none"
	case "FILTER":
		return "pending"
	default:
		return "none"
	}
}

func levelToKey(level string) string {
	switch level {
	case "CHALLENGER":
		return "challenger"
	case "GRANDMASTER":
		return "grandmaster"
	case "MASTER":
		return "master"
	case "DIAMOND":
		return "diamond"
	case "EMERALD":
		return "emerald"
	case "PLATINUM":
		return "platinum"
	case "GOLD":
		return "gold"
	case "SILVER":
		return "silver"
	case "BRONZE":
		return "bronze"
	case "IRON":
		return "iron"
	default:
		return levelToLower(level)
	}
}

func tierToLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

func levelToLower(s string) string {
	return tierToLower(s)
}
