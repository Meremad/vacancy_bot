package fetchers

import (
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func FetchFromGlassdoor(params SearchParams) []Vacancy {
	baseURL := "https://www.glassdoor.com/Job/jobs.htm"
	query := BuildQueryParams(params, "glassdoor")
	url := baseURL + "?" + query.Encode()

	resp, err := http.Get(url)
	if err != nil {
		logError("Glassdoor", "Ошибка запроса", err)
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logError("Glassdoor", "Ошибка парсинга HTML", err)
		return nil
	}

	var vacancies []Vacancy
	doc.Find(".jobLink").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		link, _ := s.Attr("href")
		vacancies = append(vacancies, Vacancy{
			Title: title,
			Link:  "https://www.glassdoor.com" + link,
		})
	})

	return vacancies
}
