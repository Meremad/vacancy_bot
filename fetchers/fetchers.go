package fetchers

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type Vacancy struct {
	Title string
	Link  string
}

type SearchParams struct {
	Language string
	City     string
	Remote   string // "online", "offline", or "both"
}

func GetCityIDFromHH(cityName string) string {
	baseURL := "https://api.hh.ru/suggests/areas"
	query := url.Values{}
	query.Add("text", cityName)

	resp, err := http.Get(baseURL + "?" + query.Encode())
	if err != nil {
		logError("HH", "Ошибка запроса ID города", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"items"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		logError("HH", "Ошибка декодирования JSON для ID города", err)
		return ""
	}

	if len(result.Items) > 0 {
		return result.Items[0].ID
	}

	return ""
}

func BuildQueryParams(params SearchParams, platform string) url.Values {
	query := url.Values{}
	if params.Language != "" {
		if platform == "hh" {
			query.Add("text", params.Language)
		} else if platform == "habr" {
			query.Add("q", params.Language)
		} else if platform == "glassdoor" {
			query.Add("sc.keyword", params.Language)
		}
	}

	if params.City != "" && params.City != "любой" {
		if platform == "hh" {
			cityID := GetCityIDFromHH(params.City)
			if cityID != "" {
				query.Add("area", cityID)
			}
		} else if platform == "glassdoor" || platform == "habr" {
			query.Add("city", params.City)
		}
	}

	if params.Remote == "online" {
		if platform == "glassdoor" {
			query.Add("remoteWorkType", "REMOTE")
		} else if platform == "hh" {
			query.Add("schedule", "remote")
		} else if platform == "habr" {
			query.Add("remote", "true")
		}
	} else if params.Remote == "offline" {
		if platform == "glassdoor" {
			query.Add("remoteWorkType", "ONSITE")
		} else if platform == "hh" {
			query.Add("schedule", "fullDay")
		} else if platform == "habr" {
			query.Add("remote", "false")
		}
	}

	return query
}

func logError(platform, message string, err error) {
	log.Printf("[%s] Ошибка: %s - %v", platform, message, err)
}

func FetchAllVacancies(params SearchParams) []Vacancy {
	var allVacancies []Vacancy
	ch := make(chan []Vacancy, 3)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- FetchFromHabr(params)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- FetchFromGlassdoor(params)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- FetchFromHeadHunter(params)
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	for vacancies := range ch {
		allVacancies = append(allVacancies, vacancies...)
	}

	return allVacancies
}
