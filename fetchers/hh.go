package fetchers

import (
	"encoding/json"
	"net/http"
)

func FetchFromHeadHunter(params SearchParams) []Vacancy {
	baseURL := "https://api.hh.ru/vacancies"
	query := BuildQueryParams(params, "hh")
	url := baseURL + "?" + query.Encode()

	resp, err := http.Get(url)
	if err != nil {
		logError("HH", "Ошибка запроса", err)
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			Name string `json:"name"`
			URL  string `json:"alternate_url"`
		} `json:"items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		logError("HH", "Ошибка декодирования JSON", err)
		return nil
	}

	var vacancies []Vacancy
	for _, item := range result.Items {
		vacancies = append(vacancies, Vacancy{
			Title: item.Name,
			Link:  item.URL,
		})
	}

	return vacancies
}
