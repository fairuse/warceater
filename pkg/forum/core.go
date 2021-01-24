package forum

type Body struct {
	Id       string
	User     string
	UserIcon string
	Hdr      string
	Msg      string
	Html     string `json:"html"`
}
