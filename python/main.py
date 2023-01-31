from arango import ArangoClient
from flask import Flask, render_template
from flask import request
import tablib
import os, glob
from bs4 import BeautifulSoup

app = Flask(__name__,
template_folder='html')

ACTIVE_SESSION_COLLECTION_NAME = "activeUsers"

# Initialize the ArangoDB client.
client = ArangoClient(hosts='http://arangodb:8529')

db = client.db('quizzer', username='quizzer', password='quizzer')

activeUserCollection = db.collection(ACTIVE_SESSION_COLLECTION_NAME)


@app.route('/')
def home():
   return render_template('index.html')

@app.route('/question/quiz/<number>')
def question(number):
   return render_template('quiz_'+number+'.html')

@app.route('/quizzer/result', methods=['GET'])
def getResultByUsername():
    username = request.args.get('username')
    try:
        user = activeUserCollection.get(username)
        results = user["results"]
    except:
        return render_template('user_not_found.html')
    processedResult = processResult(results, username)
    html = generateResultHTMLElements(username, processedResult)
    print(html, file=open('html/result.html', 'w+'))

    return render_template('result.html')

def processResult(results, username):
    for filename in glob.glob("html/quiz_*"):
        os.remove(filename) 
    totalQuizAttempted = getTotalQuizOfUser(results)
    passedQuiz = getNumberOfPassedQuiz(results)
    totalPercentage = getPercentageOfOverallPassedQuiz(passedQuiz, totalQuizAttempted)
    
    data = tablib.Dataset(headers=['Quiz Nos', 'Is Passed?', 'Nos of Correct Answers', 'Nos of Incorrect Answers', 'Total Questions','Scored Percentage', 'Goto Quiz Summary'])
    for index, result in enumerate(results):
        questions = result["questions"]
        questionsData = tablib.Dataset(headers=['Question', 'Choices', 'Correct Answer', 'User Selected Answer'])
        for index, question in enumerate(questions):   
            questionsData.append([question["question"], question["choices"], question["correctAnswer"], question["selectedAnswer"]])

        print(questionsData.export('html'), file=open('html/quiz_' + str(result["quizNo"]) + '.html', 'w+'))
        html = generateQuizHTMLElements(str(result["quizNo"]), username)
        print(html, file=open('html/quiz_' + str(result["quizNo"]) + '.html', 'w+'))
        data.append([result["quizNo"], result["passed"], result["anwsered_correctly"], result["anwsered_incorrectly"], 10,result["scored"], "<a href='http://localhost:5000/question/quiz/" + str(result["quizNo"]) + "'>Quiz " + str(result["quizNo"]) +"</a>"])
        print(data.export('html'), file=open('html/result.html', 'w+'))
    return {
        "totalQuizRound" : totalQuizAttempted,
        "totalPassedQuizRound" : passedQuiz,
        "totalPercentage" : totalPercentage 
    }
    
# Generate HTML Tags with the json
def generateResultHTMLElements(username, processedResult):
    with open("html/result.html") as fp:
        soup = BeautifulSoup(fp, 'html.parser')
        table = soup.table
        table.attrs["align"] = "center"
        htmlTag = soup.new_tag("html")
        headTag = soup.new_tag("head")
        styleTag = soup.new_tag("style")
        styleTag.string = getStyleString()
        htmlTag.append(headTag)
        headTag.append(styleTag)
        bodyTag = soup.new_tag("body")
        htmlTag.append(bodyTag)
        headerTag = soup.new_tag("h2")
        headerTag.string = "Result Page Of: " + username
        headerTag.attrs["align"] = "center"
        headerTag.attrs["style"] = "color: #399507e0"
        bodyTag.append(headerTag)
        bodyTag.append(table)
        totalTagPassedQuiz = soup.new_tag("h4")
        totalTagPassedQuiz.string = "Passed Quiz: " + str(processedResult["totalPassedQuizRound"])
        totalTagPassedQuiz.attrs["align"] = "center"
        bodyTag.append(totalTagPassedQuiz)
        totalTagAttemtedQuiz = soup.new_tag("h4")
        totalTagAttemtedQuiz.string = "Total Quiz: " + str(processedResult["totalQuizRound"])
        totalTagAttemtedQuiz.attrs["align"] = "center"
        bodyTag.append(totalTagAttemtedQuiz)
        totalTagPercentage = soup.new_tag("h4")
        totalTagPercentage.string = "Percentage: " + str(processedResult["totalPercentage"])
        totalTagPercentage.attrs["align"] = "center"
        bodyTag.append(totalTagPercentage)
        pTag = soup.new_tag("p")
        pTag.attrs["align"] = "center"
        goHomeTag = soup.new_tag("a", href="http://localhost:5000/")
        goHomeTag.string = "Go Home"
        pTag.append(goHomeTag)
        bodyTag.append(pTag)
        soup.append(htmlTag)
    return soup

# Generate HTML Tags with the json
def generateQuizHTMLElements(quizNo, username):
    with open("html/quiz_"+ quizNo +".html") as fp:
        soup = BeautifulSoup(fp, 'html.parser')
        table = soup.table
        table.attrs["align"] = "center"
        table.attrs["style"] = "width: 100%"
        htmlTag = soup.new_tag("html")
        headTag = soup.new_tag("head")
        styleTag = soup.new_tag("style")
        styleTag.string = getStyleString()
        htmlTag.append(headTag)
        headTag.append(styleTag)
        bodyTag = soup.new_tag("body")
        htmlTag.append(bodyTag)
        headerTag = soup.new_tag("h2")
        headerTag.string = "Quiz Number: " +    quizNo
        headerTag.attrs["align"] = "center"
        bodyTag.append(headerTag)
        bodyTag.append(table)
        pTag = soup.new_tag("p")
        pTag.attrs["align"] = "center"
        goBackTag = soup.new_tag("a", href="http://localhost:5000/quizzer/result?username=" + username)
        goBackTag.string = "Go Back"
        pTag.append(goBackTag)
        bodyTag.append(pTag)
        soup.append(htmlTag)
    return soup

def getTotalQuizOfUser(results):
    return len(results)

def getNumberOfPassedQuiz(results):
    passedCounter = 0
    for result in results:
        if result["passed"]:
            passedCounter = passedCounter + 1
    return passedCounter

def getPercentageOfOverallPassedQuiz(passedQuiz, totalQuiz):
    return passedQuiz * 100 / totalQuiz

def getStyleString():
    return " \n table {\n                font-family: arial, sans-serif;\n                border-collapse: collapse;\n                width: 500px;\n            }\n    \n            td, th {\n                border: 1px solid #dddddd;\n                text-align: left;\n                padding: 8px;\n            }\n    \n            tr:nth-child(even) {\n                background-color: #dddddd;\n            } \n"

app.run(debug=True, port=5000, host='0.0.0.0')