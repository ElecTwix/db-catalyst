package basic

type Posts struct {
	Id       int32
	AuthorId int32
	Title    string
	Body     string
	Status   string
}
type Users struct {
	Id        int32
	Username  string
	Email     string
	CreatedAt *interface{}
}
