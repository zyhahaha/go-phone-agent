package config

// AppPackages 应用名称到包名的映射
var AppPackages = map[string]string{
	// Social & Messaging
	"微信": "com.tencent.mm",
	"QQ":  "com.tencent.mobileqq",
	"微博": "com.sina.weibo",

	// E-commerce
	"淘宝":    "com.taobao.taobao",
	"京东":    "com.jingdong.app.mall",
	"拼多多":  "com.xunmeng.pinduoduo",

	// Lifestyle & Social
	"小红书": "com.xingin.xhs",
	"豆瓣":   "com.douban.frodo",
	"知乎":   "com.zhihu.android",

	// Maps & Navigation
	"高德地图": "com.autonavi.minimap",
	"百度地图": "com.baidu.BaiduMap",

	// Food & Services
	"美团":    "com.sankuai.meituan",
	"大众点评": "com.dianping.v1",
	"饿了么":   "me.ele",

	// Travel
	"携程":     "ctrip.android.view",
	"铁路12306": "com.MobileTicket",
	"12306":    "com.MobileTicket",
	"去哪儿":   "com.Qunar",
	"滴滴出行":  "com.sdu.didi.psnger",

	// Video & Entertainment
	"bilibili":  "tv.danmaku.bili",
	"抖音":      "com.ss.android.ugc.aweme",
	"快手":      "com.smile.gifmaker",
	"腾讯视频":   "com.tencent.qqlive",
	"爱奇艺":    "com.qiyi.video",

	// Music & Audio
	"网易云音乐": "com.netease.cloudmusic",
	"QQ音乐":   "com.tencent.qqmusic",
	"喜马拉雅":  "com.ximalaya.ting.android",

	// Reading
	"番茄小说":      "com.dragon.read",
	"番茄免费小说":   "com.dragon.read",
	"七猫免费小说":   "com.kmxs.reader",

	// Productivity
	"飞书": "com.ss.android.lark",

	// AI & Tools
	"豆包": "com.larus.nova",

	// News & Information
	"腾讯新闻":  "com.tencent.news",
	"今日头条":  "com.ss.android.article.news",

	// System apps
	"Settings": "com.android.settings",
	"Chrome":  "com.android.chrome",
}

// GetPackageName 获取应用包名
func GetPackageName(appName string) (string, bool) {
	pkgName, ok := AppPackages[appName]
	return pkgName, ok
}

// ListSupportedApps 返回所有支持的应用列表
func ListSupportedApps() []string {
	apps := make([]string, 0, len(AppPackages))
	for app := range AppPackages {
		apps = append(apps, app)
	}
	return apps
}
