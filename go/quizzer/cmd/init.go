/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"quizzer/models"
	"time"

	driver "github.com/arangodb/go-driver"
	arangohttp "github.com/arangodb/go-driver/http"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var OPENTDB_API_BASE_URL string = "https://opentdb.com/"
var OPENTDB_API_QUESTIONS_AMOUNT string = "10"
var ACTIVE_SESSION_COLLECTION_NAME = "activeUsers"


// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the quizzer game for the user.",
	Long: `The CLI tool is initialized to start a session with a user to play a round of game. This is will start a user interactive session.
	If the user leaves (forcefully quits) the game, then no results will be captured for that left game and will also not be reflected in the 
	overall result for that user.`,
	Run: func(cmd *cobra.Command, args []string) {
		
		// Your business logic
		fmt.Println("Welcome to the Quizzer Tool. In order to start the game, please enter your inGame name below : ")
		fmt.Println("Type Your inGame Name :")
		var inGameName = getScanText() // search in inGame name collection that if this username exists. If not exist, then create a new entry as User models.
	
		// check if document exist in the activeUsers collection
		var client = databaseConnectionClient() // Getting Client
		var db = openDatabase("quizzer", client) // Getting DB object
		
		// Create collections "activeUsers" and "results" for the very first time if not present
		checkAndCreateCollectionIfNotPresent(ACTIVE_SESSION_COLLECTION_NAME, db)

		var token = processActiveUserInSession(inGameName, ACTIVE_SESSION_COLLECTION_NAME, db) // Process active user and get session token accordingly
		resp := pullQuestions(inGameName, token, db) // Request OpenTDB to request Question sets
	
		
		//Quiz Starts Here
		resultObject := processQuizAction(resp, inGameName, db)
		replaceDocumentOfUserForResults(resultObject, inGameName, db)
		
		var testStatus string
		var symbol string
		if resultObject.Passed {
			testStatus = "PASSED"
			symbol = ":-)"
		} else {
			testStatus = "FAILED"
			symbol = ":-("
		}

		fmt.Printf("You %s the Quiz %s", testStatus, symbol)
		fmt.Println("")
		fmt.Printf("You Scored: %d percent", resultObject.Scored)
		fmt.Println("")
		fmt.Printf("Out of 10 questions, you answered %d correctly and %d incorrectly", resultObject.AnwseredCorrectly, resultObject.AnwseredIncorrectly)
		fmt.Println("")
		fmt.Println("")
		fmt.Println("Summary of the Quiz")
		for i, q := range resultObject.Questions {
			fmt.Printf("Question %d: %s", i+1, q.Question)
			fmt.Println("")
			fmt.Printf("Your Selected Answer: %s", q.Selectedanswer)
			fmt.Println("")
			fmt.Printf("Correct Answer: %s", q.Correctanswer)
			fmt.Println("")
			fmt.Println("-----------------------------------------------")
			fmt.Println("")
		}
	},
}


func processQuizAction(results models.Response, key string, db driver.Database) models.Result  {
	resultSet := make(map[int]int)
	var questions []models.Question
	for i := 0; i < len(results.Results); i++ {
		var size int
		if stringToBase64Decode(results.Results[i].Type) == "multiple" {
			size = 4
		} else {
			size = 2
		}

		allChoices := append(sliceToBase64Decode(results.Results[i].IncorrectAnswers), stringToBase64Decode(results.Results[i].CorrectAnswer))
		shuffleSlice(allChoices)
		selectedAnswer := promptSelector(
			size, 
			allChoices,
			stringToBase64Decode(results.Results[i].Question),
		)

		if selectedAnswer == stringToBase64Decode(results.Results[i].CorrectAnswer) {
			resultSet[i+1] = 1
		} else {
			resultSet[i+1] = 0
		}

		question := models.Question{
			Question: stringToBase64Decode(results.Results[i].Question),
			Choices: allChoices,
			Correctanswer: stringToBase64Decode(results.Results[i].CorrectAnswer),
			Selectedanswer: selectedAnswer,
		}
		questions = append(questions, question)
	}

	doc := readUserFromDocumentKey(key, db)


	var quizNo int
	if len(doc.Results) != 0 {
		quizNo = doc.Results[len(doc.Results) - 1].Quizno + 1
	} else {
		quizNo = len(doc.Results) + 1
	}


	return models.Result{
		Passed: isPassed(addSumOfValuesInMap(resultSet)),
		Scored: addSumOfValuesInMap(resultSet) * 100 / 10,
		Quizno: quizNo,
		AnwseredIncorrectly: len(results.Results) - addSumOfValuesInMap(resultSet),
		AnwseredCorrectly: addSumOfValuesInMap(resultSet),
		Questions: questions,
	}
}

func shuffleSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func addSumOfValuesInMap(resultMap map[int]int) int {
	sum := 0
	for _, v := range resultMap {
		sum += v
	}
	return sum
}

func readUserFromDocumentKey(key string, db driver.Database) models.User {
	var doc models.User
	col := openCollection(ACTIVE_SESSION_COLLECTION_NAME, db)
	_, err := col.ReadDocument(context.Background(), key, &doc)
	if err != nil {
		log.Fatalf("Document not found with name %s. %s", key, err)
		os.Exit(1)
	}
	return doc
}

// Anything 60 and above percentage is considered as PASSED
func isPassed(scoredPoints int) bool {
	return scoredPoints * 100 / 10 >= 60
}

// Create new entry for the user who came very first time
func generateNewDocumentForActiveSession(token string, key string, col driver.Collection) string {
	ctx := context.Background()

	session := models.Session{
		Token: token,
		Time: time.Now().UnixMilli(),
	}

	var sessions = []models.Session{};
	
	sessions = append(sessions, session)
	
	users := models.User{
		Key: key,
		Sessions: sessions,
	}

	_, err := col.CreateDocument(ctx, users)
	if err != nil {
		log.Fatalf("Could not create document. %s", err)
	}
	return users.Sessions[len(users.Sessions)-1].Token
}

func replaceDocumentOfUserForResults(result models.Result, key string, db driver.Database)  {
	user := readUserFromDocumentKey(key, db)
	user.Results = append(user.Results, result)
	
	col := openCollection(ACTIVE_SESSION_COLLECTION_NAME, db)
	_, err := col.ReplaceDocument(context.Background(), key, user)
	if err != nil {
		log.Fatalf("Could not replace document. %s", err)
		os.Exit(1)
	}

}

// Update entry for user existing in the system for active sessions accordingly
func replaceExistingDocumentForActiveSession(user models.User, token string, key string, col driver.Collection) string  {
	ctx := context.Background()
	var session models.Session
	var previousTime = user.Sessions[len(user.Sessions)-1].Time
	if isGivenTimeBeforeSixHours(previousTime) {
		session = models.Session{
			Token: user.Sessions[len(user.Sessions)-1].Token,
			Time: time.Now().UnixMilli(),
		}
	} else {	// Token should also be reset or request when question is exhausted with the same token
		session = models.Session{ 
			Token: token,
			Time: time.Now().UnixMilli(),
		}
	}
	user.Sessions = append(user.Sessions, session)
	_, err := col.ReplaceDocument(ctx, key, user)
	if err != nil {
		log.Fatalf("Could not replace document. %s", err)
		os.Exit(1)
	}
	return user.Sessions[len(user.Sessions)-1].Token
}

func openCollection(collectionName string, db driver.Database) driver.Collection {
	ctx := context.Background()
	col, err := db.Collection(ctx, collectionName)
	if err != nil {
		log.Fatalf("Error occured while opening the connection with name %s", collectionName)
		os.Exit(1) 
	}
	return col
}

func processActiveUserInSession(inGameName string, collectionName string, db driver.Database) string {
	var doc models.User
	col := openCollection(collectionName, db)
	ctx := context.Background()
	_, err := col.ReadDocument(ctx, inGameName, &doc)
	if err != nil {
		log.Printf("Document not found with name %s", inGameName)
		return generateNewDocumentForActiveSession(generateOpenTDBSessionToken(), inGameName, col)
	}

	return replaceExistingDocumentForActiveSession(doc, generateOpenTDBSessionToken(), inGameName, col)
}

func stringToBase64Decode(content string) string {
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
			log.Fatal("error:", err)
	}
	return string(data)
}

func sliceToBase64Decode(contents []string) []string {
	var decodedList []string
	for _, content := range contents {
		decodedList = append(decodedList, stringToBase64Decode(content))
	}
	return decodedList
}

/* func formatTime(time time.Time) string {
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d",
        time.Year(), time.Month(), time.Day(),
        time.Hour(), time.Minute())
	return formatted
} */

//Generate or pull Questions from OpenTB and Reset session if required
func pullQuestions(key string, token string, db driver.Database) models.Response {
	EXTENDED_URL := "api.php?amount=" + OPENTDB_API_QUESTIONS_AMOUNT + "&encode=base64&token=" + token
	var resp string = get(EXTENDED_URL)
	var respDetails models.Response
	json.Unmarshal([]byte(resp), &respDetails)
	if respDetails.ResponseCode == 3 || respDetails.ResponseCode == 4  {
		// reset and update the token for the given user
		newToken := generateOpenTDBSessionToken()
		updateDocumentWithNewToken(key, newToken, db)
		return pullQuestions(key, newToken, db)
	}
	return respDetails
}

func updateDocumentWithNewToken(key string, token string, db driver.Database)  {
	var doc models.User
	col := openCollection(ACTIVE_SESSION_COLLECTION_NAME, db)
	session := models.Session{
		Token: token,
		Time: time.Now().UnixMilli(),
	}
	_, err := col.ReadDocument(context.Background(), key, &doc)
	if err != nil {
		log.Fatalf("Document not found with name %s. %s", key, err)
		os.Exit(1)
	}
	doc.Sessions = doc.Sessions[:len(doc.Sessions) - 1]
	doc.Sessions = append(doc.Sessions, session)
	col.UpdateDocument(context.Background(), key, doc)
}

//Generate Session Token to not repeat questions
func generateOpenTDBSessionToken() string {
	EXTENDED_URL := "api_token.php?command=request"
	var token string = get(EXTENDED_URL)
	var tokenDetails models.Token
	json.Unmarshal([]byte(token), &tokenDetails)
	return tokenDetails.Token
}

// GET REST Client
func get(EXTENDED_URL string) string {
	resp, err := http.Get(OPENTDB_API_BASE_URL + EXTENDED_URL)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	return sb
}

// Session tokens expires in 6 hours of inactivity
func isGivenTimeBeforeSixHours(givenTime int64) bool {
	timeDifferInMillis := time.Now().UnixMilli() - givenTime
	timeDifferInHours := float64(timeDifferInMillis) / float64(1000 * 60 * 60)
	return timeDifferInHours < 6
}



func checkAndCreateCollectionIfNotPresent(activeUsers string, db driver.Database)  {
	if !collectionExists(activeUsers, db) {
		createCollection(activeUsers, db)
	}
}

func collectionExists(collectionName string, db driver.Database) bool {
	ctx := context.Background()
	found, err := db.CollectionExists(ctx, collectionName)
	if err != nil {
		log.Fatalf("Cannot read collection with name %s", collectionName)
		os.Exit(1)
	}
	return found
}

func createCollection(collectionName string, db driver.Database)  {
	ctx := context.Background()
	options := driver.CreateCollectionOptions{ /* ... */ }
	col, err := db.CreateCollection(ctx, collectionName, &options)
	if err != nil {
		log.Fatalf("Cannot read collection with name %s. %s", collectionName, err)
		os.Exit(1)
	}
	_ = col
}

func openDatabase(name string, client driver.Client)  driver.Database {
	ctx := context.Background()
	db, err := client.Database(ctx, name)
	if err != nil {
		log.Fatalf("Cannot open database with name %s. %s", name, err)
		os.Exit(1)
	}
	return db
}

func databaseConnectionClient() driver.Client {
	var err error
	var client driver.Client
	var conn   driver.Connection

	// Open a client connection 
	conn, err = arangohttp.NewConnection(arangohttp.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529/"},
	})

	if err != nil {
		log.Fatalf("Cannot reach to the Database with the provided hostname and port.")
		os.Exit(1)
	}

	// Client object
	client, err = driver.NewClient(driver.ClientConfig{
		Connection: conn,
		Authentication: driver.BasicAuthentication("quizzer", "quizzer"),
	})
	if err != nil {
		log.Fatalf("Cannot connect to the Database. %s", err)
		os.Exit(1)
	}
	return client
}

func getScanText() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

func promptSelector(size int, elements []string, label string) string {
	prompt := promptui.Select{ // Displays an interactive select list tool
		Label: label,
		Size:  size,
		Items: elements,
		Templates: &promptui.SelectTemplates{
			Active:   ` ✅ {{ . | cyan | bold }}`,
			Inactive: `   {{ . | cyan }}`,
			Selected: `{{ "✔" | green | bold }} {{ "You Selected: " | bold }}: {{ . | cyan }}`,
		},
	}
	_, res, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "Something failed while initializing PromptUI Select."
	}
	return res
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
