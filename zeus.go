package zeus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"appengine"
	"appengine/urlfetch"
)

// API constants
const (
	APIKey string = "a7a529651ced685c47027f22a11c0029f32fe16c"
	APIURL string = "http://api.worldweatheronline.com/free/v1/weather.ashx"
)

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/weather", weather)
}

// WeatherValue is a struct for decoding the json responses from World Weather Online API.
// Only the required fields are added and the rest of them are ignored automatically by the decoder.
type WeatherValue struct {
	Data struct {
		Request []struct {
			City string `json:"query"`
		} `json:"request"`
		Weather []struct {
			MinC string `json:"tempMinC"`
			MaxC string `json:"tempMaxC"`
		}
		Error []struct {
			Msg string `json:"msg"`
		} `json:"error"`
	} `json:"data"`
}

var templates = template.Must(template.ParseFiles("form.html")) // Add more templates after form.html separated by a comma

func handler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "form.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func weather(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var a []WeatherValue
	for _, city := range r.Form {
		x, err := query(ctx, city[0])

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		a = append(a, x)
	}
	fmt.Fprint(w, a)
}

func query(ctx appengine.Context, city string) (WeatherValue, error) {
	client := urlfetch.Client(ctx)
	var wv WeatherValue

	v := url.Values{}
	v.Set("q", city)
	v.Add("format", "json")
	v.Add("key", APIKey)

	cityURL, err := url.Parse(APIURL + "?" + v.Encode())
	if err != nil {
		return wv, err
	}

	resp, err := client.Get(cityURL.String())
	if err != nil {
		return wv, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&wv); err != nil {
		return wv, err
	}

	return wv, nil
}
