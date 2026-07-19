package web

const SiteBaseURL = "https://cairn.cn"

func CanonicalURL(path string) string {
	return SiteBaseURL + path
}
