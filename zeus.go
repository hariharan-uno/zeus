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
	http.HandleFunc("/", formHandler)
	http.HandleFunc("/weather", weatherHandler)
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

func formHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "form.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	values := batchQuery(ctx, r.Form)
	fmt.Fprint(w, values)
}

// batchQuery queries multiple cities obtained from url values concurrently
// and returns a slice of WeatherValues.
func batchQuery(ctx appengine.Context, f url.Values) []WeatherValue {
	out := make(chan WeatherValue)
	for _, v := range f {
		// spawn a goroutine on an anonymous function to query the data of a single city.
		go func(city string) {
			wv, err := query(ctx, city)
			if err != nil {
				// Here we just log the error as an empty WeatherValue variable
				// will be shown as an error in the template.
				ctx.Errorf("%s", err)
				out <- wv
			}
			out <- wv
		}(v[0]) // we pass v[0] because url.Values is map[string][]string.
	}
	var result []WeatherValue
	for i := 0; i < len(f); i++ {
		result = append(result, <-out)
	}
	return result
}

// query fetches the WeatherValue of a single city by querying the
// World Weather Online API.
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
