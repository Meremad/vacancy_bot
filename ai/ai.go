package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Assistant struct{}

// NewAssistant создает новый экземпляр Assistant
func NewAssistant() *Assistant {
	return &Assistant{}
}

// RequestBody представляет структуру входящего JSON-запроса
type RequestBody struct {
	Question string `json:"question"`
}

// ResponseBody представляет структуру исходящего JSON-ответа
type ResponseBody struct {
	Answer string `json:"answer"`
	Error  string `json:"error,omitempty"`
}

func (a *Assistant) GetResponse(prompt string) (string, error) {
	// Добавляем инструкции по форматированию в промпт
	formattedPrompt := fmt.Sprintf(`
Ответь на следующий вопросты senior програмист и у тебя справшиват новичок в этом , используя форматирование для Telegram:

- Используй `+"```"+` для блоков кода
- Используй • для маркированных списков
- Добавляй пустые строки между параграфами и секциями для лучшей читаемости
- также можешь добавить смайлики в ответах для более дружелюбного ответа и более человеческого общения

Вопрос: %s

Пожалуйста, дай хороший ответ.`, prompt)

	response, err := AskAI(formattedPrompt)
	if err != nil {
		log.Printf("Ошибка в AI: %v", err)
		return "", errors.New("произошла ошибка при обработке запроса")
	}
	return response, nil
}

// AskAI отправляет запрос к локальному Python серверу и возвращает ответ
func AskAI(question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", errors.New("no question provided")
	}

	apiURL := "http://localhost:5001/ask"
	payload := RequestBody{
		Question: question,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling payload: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Python server: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("python server error: %s", string(body))
	}

	var responseBody ResponseBody
	if err := json.Unmarshal(body, &responseBody); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %v", err)
	}

	if responseBody.Error != "" {
		return "", fmt.Errorf("error from Python server: %s", responseBody.Error)
	}

	return responseBody.Answer, nil
}
