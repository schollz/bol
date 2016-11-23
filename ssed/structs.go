package ssed

type config []struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Method string `json:"method"`
}

type entry struct {
	Text string `json:"text"`
	Timestamp string `json:"timestamp"`
	Document string `json:"document"`
	Entry string `json:"entry"`
}

type document struct {
  Entries []Entry
}
