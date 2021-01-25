package forum

type Body struct {
	Id       string `json:"id"`
	User     string `json:"user"`
	UserIcon string `json:"usericon"`
	Hdr      string `json:"hdr"`
	Msg      string `json:"msg"`
	Html     string `json:"html"`
}
