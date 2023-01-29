package models

type User struct {
	Key     	string 		`json:"_key"`
	Sessions 	[]Session	`json:"sessions"`
	Results 	[]Result	`json:"results"`
}

type Session struct {
	Token string `json:"token"`
	Time  int64 `json:"time"`
}


type Result struct {
	Quizno              int  			`json:"quizNo"`
	Passed              bool 			`json:"passed"`
	AnwseredCorrectly   int  			`json:"anwsered_correctly"`
	AnwseredIncorrectly int  			`json:"anwsered_incorrectly"`
	Scored              int  			`json:"scored"`
	Questions 			[]Question 		`json:"questions"`
}

type Question struct {
	Question      string   `json:"question"`
	Choices       []string `json:"choices"`
	Correctanswer string   `json:"correctAnswer"`
	Selectedanswer string   `json:"selectedAnswer"`
}
