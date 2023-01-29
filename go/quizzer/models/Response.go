package models

type Response struct {
	ResponseCode int 	`json:"response_code"`
	Results      []Res 	`json:"results"`
}

type Res struct {
	Category         string   `json:"category"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}