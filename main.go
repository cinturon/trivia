package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"html"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list            list.Model
	choice          string
	starting        bool
	quitting        bool
	currentQuestion int
	totalQuestions  int
	score           int
	correctAnswer   string
	quiz            Quiz
}
type Quiz struct {
	ResponseCode int       `json:"response_code"`
	Results      []Results `json:"results"`
}

type Results struct {
	Category         string   `json:"category"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

func QuestionsMsg() tea.Msg {
	url := "https://opentdb.com/api.php?amount=50&category=11"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching questions:", err)
	}
	defer resp.Body.Close()
	var quiz Quiz
	err = json.NewDecoder(resp.Body).Decode(&quiz)
	if err != nil {
		fmt.Println("Error decoding question:", err)
	}
	return quiz
}

func (m model) Init() tea.Cmd {
	return QuestionsMsg
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case Quiz:
		m.quiz = msg

		return m, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "s":
			m.starting = true
			createQuestion(&m)

		case "enter":
			if m.starting == false {
				return m, nil
			}
			i, ok := m.list.SelectedItem().(item)
			if ok {
				if m.correctAnswer == string(i) {
					m.score++
				}
			}
			if m.currentQuestion < len(m.quiz.Results) {
				createQuestion(&m)
			} else {
				m.currentQuestion = 0
				return m, QuestionsMsg
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func createQuestion(m *model) {
	m.correctAnswer = m.quiz.Results[m.currentQuestion].CorrectAnswer
	m.totalQuestions++
	m.list.Title = fmt.Sprintf("Question #%d - Score:%d \n%s", m.totalQuestions, m.score, html.UnescapeString(m.quiz.Results[m.currentQuestion].Question))
	m.correctAnswer = m.quiz.Results[m.currentQuestion].CorrectAnswer

	var choices []string
	choices = append(choices, m.quiz.Results[m.currentQuestion].IncorrectAnswers...)
	choices = append(choices, m.quiz.Results[m.currentQuestion].CorrectAnswer)

	// Shuffle the slice

	rand.Shuffle(len(choices), func(i, j int) {
		choices[i], choices[j] = choices[j], choices[i]
	})
	var items []list.Item
	for _, choice := range choices {
		choice = html.UnescapeString(choice)
		fmt.Println(choice)
		items = append(items, item(choice))
	}

	m.list.SetItems(items)
	m.currentQuestion++
}

func (m model) View() string {
	if m.starting == false {
		return fmt.Sprintf("Press 's' To Start Quiz")
	}
	if m.quitting {
		return quitTextStyle.Render("Game Over! Your score is: " + fmt.Sprint(m.score))
	}
	return "\n" + m.list.View()
}

func main() {

	const defaultWidth = 300

	l := list.New([]list.Item{}, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
