package azdo

type projectResponseEnvelope struct {
	Count int `json:"count"`
	Projects  []Project `json:"value"`
}

type Project struct {
	Id   string    `json:"id"`
	Name string    `json:"name"`
	Builds []Build
}
