package fetchers

import (
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func FetchFromHabr(params SearchParams) []Vacancy {
	baseURL := "https://career.habr.com/vacancies"
	query := BuildQueryParams(params, "habr")
	url := baseURL + "?" + query.Encode()

	resp, err := http.Get(url)
	if err != nil {
		logError("Habr", "Ошибка запроса", err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logError("Habr", "Ошибка парсинга HTML", err)
		return nil
	}

	var vacancies []Vacancy
	doc.Find(".vacancy-card__title-link").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		link, _ := s.Attr("href")
		vacancies = append(vacancies, Vacancy{
			Title: title,
			Link:  "https://career.habr.com" + link,
		})
	})

	return vacancies
}
