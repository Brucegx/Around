package main

import (
	elastic "gopkg.in/olivere/elastic.v3"
	"fmt"
	"net/http"
	"encoding/json"
	"log"
	"strconv"
	"reflect"
	"context"
	"io"
	"time"
    "cloud.google.com/go/bigtable"
	"strings"
	//"os"
	//"github.com/rs/cors"
	"github.com/pborman/uuid"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	//"github.com/gorilla/handlers"
	"github.com/go-redis/redis"
	"cloud.google.com/go/storage"
)

const (
	INDEX = "around"
	TYPE = "post"
	DISTANCE = "200km"
	PROJECT_ID = "around-217516"
	BT_INSTANCE = "around-post"
	ENABLE_MEMCACHE = true
	ENABLE_BIGTABLE = false
	ES_URL = "http://35.229.78.216:9200/"
	//BUCKET_NAME = "post-image-217516"
	REDIS_URL = "redis-13261.c14.us-east-1-2.ec2.cloud.redislabs.com:13261"
	REDIS_PASSWORD = "zXutc9MScjWHbSSOc1Y4ubIZc9HaYk0E"
)


type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct {
	User string `json:"user"`
	Message string `json:"message"`
	Location Location `json:"location"`
	Url string `json:"url"`
}

var (
	mySigningKey        = []byte("secret")
	BIGTABLE_PROJECT_ID = "around-217516"
	GCS_BUCKET          = "post-image-217516"
)

func main() {
	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(INDEX).Do()
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		mapping := `{
                    "mappings":{
                           "post":{
                                  "properties":{
                                         "location":{
                                                "type":"geo_point"
                                         }
                                  }
                           }
                    }
             }
             `
		_, err := client.CreateIndex(INDEX).Body(mapping).Do()
		if err != nil {
			// Handle error
			panic(err)
		}
	}

	fmt.Println("Started service successfully")
	r := mux.NewRouter()
	
	var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return mySigningKey, nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
	// headersOk := handlers.AllowedHeaders([]string{"X-Requested-With","Content-Type"})
	// originsOk := handlers.AllowedOrigins([]string{"Access-Control-Allow-Origin"})
	// methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	// // start server listen
	// // with error handling
	
	// http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(r))

	r.Handle("/post", jwtMiddleware.Handler(http.HandlerFunc(handlerPost)))
	r.Handle("/search", jwtMiddleware.Handler(http.HandlerFunc(handlerSearch)))
	r.Handle("/login", http.HandlerFunc(loginHandler)).Methods("POST")
	r.Handle("/signup", http.HandlerFunc(signupHandler)).Methods("POST")

	// c := cors.New(cors.Options{
	// 	AllowCredentials: true,
	// 	OptionsPassthrough: true,
    // })

    // handler := c.Handler(r)
	// http.ListenAndServe(":8080", handler)
	
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}


func handlerPost (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method != "POST" {
		return
	}

	user := r.Context().Value("user")
	if user == nil {
		m := fmt.Sprintf("Unable to find user in context")
		fmt.Println(m)
		http.Error(w, m, http.StatusBadRequest)
		return
	}
	claims := user.(*jwt.Token).Claims
	username := claims.(jwt.MapClaims)["username"]

	// 32 << 20 is the maxMemory param for ParseMultipartForm
	// After you call ParseMultipartForm, the file will be saved in the server memory with maxMemory size.
	// If the file size is larger than maxMemory, the rest of the data will be saved in a system temporary file.
	r.ParseMultipartForm(32 << 20)

	// Parse from form data.
	fmt.Printf("Received one post request %s\n", r.FormValue("message"))
	lat, _ := strconv.ParseFloat(r.FormValue("lat"), 64)
	lon, _ := strconv.ParseFloat(r.FormValue("lon"), 64)
	p := &Post{
		User:    username.(string),
		Message: r.FormValue("message"),
		Location: Location{
			Lat: lat,
			Lon: lon,
		},
	}

	id := uuid.New()

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image is not available", http.StatusInternalServerError)
		fmt.Printf("Image is not available %v.\n", err)
		return
	}

	ctx := context.Background()

	defer file.Close()
	_, attrs, err := saveToGCS(ctx, file, GCS_BUCKET, id)
	if err != nil {
		http.Error(w, "GCS is not setup", http.StatusInternalServerError)
		fmt.Printf("GCS is not setup %v\n", err)
		return
	}

	// Update the media link after saving to GCS.
	p.Url = attrs.MediaLink

	// Save to ES.
	go saveToES(p, id)

	// Save to BigTable.
	if ENABLE_BIGTABLE {
		go saveToBigTable(p, id)
	}
}

func saveToGCS(ctx context.Context, r io.Reader, bucket, name string) (*storage.ObjectHandle, *storage.ObjectAttrs, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
			return nil, nil, err
	}
	defer client.Close()

	bh := client.Bucket(bucket)
	// Next check if the bucket exists
	if _, err = bh.Attrs(ctx); err != nil {
			return nil, nil, err
	}

	obj := bh.Object(name)
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
			return nil, nil, err
	}
	if err := w.Close(); err != nil {
			return nil, nil, err
	}

	// set access all user read premission 
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			return nil, nil, err
	}

	attrs, err := obj.Attrs(ctx)
	fmt.Printf("Post is saved to GCS: %s\n", attrs.MediaLink)
	return obj, attrs, err
}

// Save a post to ElasticSearch
func saveToES(p *Post, id string) {
	// Create a client
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Save it to index
	_, err = es_client.Index().
		Index(INDEX).
		Type(TYPE).
		Id(id).
		BodyJson(p).
		Refresh(true).
		Do()
	if err != nil {
		panic(err)
		return
	}

	fmt.Printf("Post is saved to Index: %s\n", p.Message)
}

// Save a post to BigTable
func saveToBigTable(p *Post, id string) {
	ctx := context.Background()
	// update project name here
	bt_client, err := bigtable.NewClient(ctx, PROJECT_ID, BT_INSTANCE)
	if err != nil {
		panic(err)
		return
	}
	tbl := bt_client.Open("post")
	mut := bigtable.NewMutation()
	t := bigtable.Now()
	mut.Set("post", "user", t, []byte(p.User))
	mut.Set("post", "message", t, []byte(p.Message))
	mut.Set("location", "lat", t, []byte(strconv.FormatFloat(p.Location.Lat, 'f', -1, 64)))
	mut.Set("location", "lon", t, []byte(strconv.FormatFloat(p.Location.Lon, 'f', -1, 64)))
	// erro handling for debug
	err = tbl.Apply(ctx, id, mut)
	if err != nil {
		panic(err)
		return
	}
	fmt.Printf("Post is saved to BigTable: %s\n", p.Message)
}


func handlerSearch (w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one request for search")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method != "GET" {
		return
	}

	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)

	// range is optional
	ran := DISTANCE
	if val := r.URL.Query().Get("range"); val != "" {
		ran = val + "km"
	}

	key := r.URL.Query().Get("lat") + ":" + r.URL.Query().Get("lon") + ":" + ran
	if ENABLE_MEMCACHE {
		rs_client := redis.NewClient(&redis.Options{
			Addr:     REDIS_URL,
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		val, err := rs_client.Get(key).Result()
		if err != nil {
			fmt.Printf("Redis cannot find the key %s as %v.\n", key, err)
		} else {
			fmt.Printf("Redis find the key %s.\n", key)
			w.Write([]byte(val))
			return
		}
	}

	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		http.Error(w, "ES is not setup", http.StatusInternalServerError)
		fmt.Printf("ES is not setup %v\n", err)
		return
	}

	// Define geo distance query as specified in
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.2/query-dsl-geo-distance-query.html
	q := elastic.NewGeoDistanceQuery("location")
	q = q.Distance(ran).Lat(lat).Lon(lon)

	// Some delay may range from seconds to minutes. So if you don't get enough results. Try it later.
	searchResult, err := client.Search().
		Index(INDEX).
		Query(q).
		Pretty(true).
		Do()
	if err != nil {
		// Handle error
		m := fmt.Sprintf("Failed to search ES %v", err)
		fmt.Println(m)
		http.Error(w, m, http.StatusInternalServerError)
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)
	// TotalHits is another convenience function that works even when something goes wrong.
	fmt.Printf("Found a total of %d post\n", searchResult.TotalHits())

	// Each is a convenience function that iterates over hits in a search result.
	// It makes sure you don't need to check for nil values in the response.
	// However, it ignores errors in serialization.
	var typ Post
	var ps []Post
	for _, item := range searchResult.Each(reflect.TypeOf(typ)) {
		p := item.(Post)
		fmt.Printf("Post by %s: %s at lat %v and lon %v\n", p.User, p.Message, p.Location.Lat, p.Location.Lon)
		// TODO: Perform filtering based on keywords such as web spam etc.
		if !containsFilteredWords(&p.Message) {
            ps = append(ps, p)
        }
	}
	js, err := json.Marshal(ps)
	if err != nil {
		m := fmt.Sprintf("Failed to parse post object %v", err)
		fmt.Println(m)
		http.Error(w, m, http.StatusInternalServerError)
		return
	}

	if ENABLE_MEMCACHE {
		rs_client := redis.NewClient(&redis.Options{
			Addr:     REDIS_URL,
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		// Set the cache expiration to be 10 seconds
		err := rs_client.Set(key, string(js), time.Second*10).Err()
		if err != nil {
			fmt.Printf("Redis cannot save the key %s as %v.\n", key, err)
		}

	}
	w.Write(js)

}

func containsFilteredWords(s *string) bool {
        filteredWords := []string{
                "fuck",
                "1000",
        }
        for _, word := range filteredWords {
                if strings.Contains(*s, word) {
                        return true
                }
        }
        return false
}



