//go:generate statik -src=./static

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"net/url"

	"github.com/gorilla/mux"
	pilosa "github.com/pilosa/go-pilosa"
	// ssb "github.com/pilosa/pdk/ssb"
	"github.com/spf13/pflag"
)

const host = "node0.bio6.sandbox.pilosa.com:10101"
const indexName = "ssb"

var Version = "v0.0.0" // demo version

var yearMap = map[int]int{
	1992: 1992,
	1993: 1993,
	1994: 1994,
	1995: 1995,
	1996: 1996,
	1997: 1997,
	1998: 1998,
}

var regions = map[string]int{
	"AMERICA":     0,
	"AFRICA":      1,
	"ASIA":        2,
	"EUROPE":      3,
	"MIDDLE EAST": 4,
}

var americaNations = []string{"CANADA", "ARGENTINA", "BRAZIL", "UNITED STATES", "PERU"}
var asiaNations = []string{"INDIA", "INDONESIA", "CHINA", "VIETNAM", "JAPAN"}

// 5 nations per region, in same order as above
var nations = map[string]int{
	"CANADA":         0,
	"ARGENTINA":      1,
	"BRAZIL":         2,
	"UNITED STATES":  3,
	"PERU":           4,
	"ETHIOPIA":       5,
	"ALGERIA":        6,
	"KENYA":          7,
	"MOZAMBIQUE":     8,
	"MOROCCO":        9,
	"INDIA":          10,
	"INDONESIA":      11,
	"CHINA":          12,
	"VIETNAM":        13,
	"JAPAN":          14,
	"ROMANIA":        15,
	"RUSSIA":         16,
	"FRANCE":         17,
	"UNITED KINGDOM": 18,
	"GERMANY":        19,
	"SAUDI ARABIA":   20,
	"JORDAN":         21,
	"IRAN":           22,
	"IRAQ":           23,
	"EGYPT":          24,
}

var cities = make(map[string]int)
var cityIDs = make(map[int]string)

func DefineCityMap() {
	for nation, nationID := range nations {
		for j := 0; j < 10; j++ {
			cityname := fmt.Sprintf("%s%d", PadRight(nation, " ", 9)[0:9], j)
			cityID := nationID*10 + j
			cities[cityname] = cityID
			cityIDs[cityID] = cityname
			cityID += 1
		}
	}
}

func PadRight(str, pad string, length int) string {
	for {
		str += pad
		if len(str) > length {
			return str[0:length]
		}
	}
}

func main() {
	DefineCityMap()
	//translator = ssb.NewTranslator("ssdbmapping")
	//fmt.Println(translator)
	//fmt.Println(translator.Get("c_city", 0))
	//return
	pilosaAddr := pflag.String("pilosa", host, "host:port for pilosa")
	pflag.Parse()

	server, err := NewServer(*pilosaAddr)
	if err != nil {
		log.Fatalf("getting new server: %v", err)
	}
	fmt.Printf("lineorder count: %d\n", server.NumLineOrders)
	server.Serve()
}

type Server struct {
	Router        *mux.Router
	Client        *pilosa.Client
	Index         *pilosa.Index
	Frames        map[string]*pilosa.Frame
	NumLineOrders uint64
}

func (s *Server) HandleTopN(w http.ResponseWriter, r *http.Request) {
	// sanity check function
	q := `TopN(frame=lo_year)`
	fmt.Println(q)
	response, err := s.Client.Query(s.Index.RawQuery(q), nil)
	if err != nil {
		fmt.Printf("%v failed with: %v\n", q, err)
		return
	}
	for a, b := range response.Results()[0].CountItems {
		fmt.Printf("%v %v\n", a, b)
	}
}

func NewServer(pilosaAddr string) (*Server, error) {
	server := &Server{
		Frames: make(map[string]*pilosa.Frame),
	}

	router := mux.NewRouter()
	router.HandleFunc("/version", server.HandleVersion).Methods("GET")
	router.HandleFunc("/query/topn", server.HandleTopN).Methods("GET")
	router.HandleFunc("/query/1.1", server.HandleQuery11).Methods("GET")
	router.HandleFunc("/query/1.2", server.HandleQuery12).Methods("GET")
	router.HandleFunc("/query/1.3", server.HandleQuery13).Methods("GET")
	router.HandleFunc("/query/2.1", server.HandleQuery21).Methods("GET")
	router.HandleFunc("/query/2.2", server.HandleQuery22).Methods("GET")
	router.HandleFunc("/query/2.3", server.HandleQuery23).Methods("GET")
	router.HandleFunc("/query/3.1", server.HandleQuery31).Methods("GET")
	router.HandleFunc("/query/3.2", server.HandleQuery32).Methods("GET")
	router.HandleFunc("/query/3.3", server.HandleQuery33).Methods("GET")
	router.HandleFunc("/query/3.4", server.HandleQuery34).Methods("GET")
	router.HandleFunc("/query/4.1", server.HandleQuery41).Methods("GET")
	router.HandleFunc("/query/4.2", server.HandleQuery42).Methods("GET")
	router.HandleFunc("/query/4.3", server.HandleQuery43).Methods("GET")

	pilosaURI, err := pilosa.NewURIFromAddress(pilosaAddr)
	if err != nil {
		return nil, err
	}
	client := pilosa.NewClientWithURI(pilosaURI)
	index, err := pilosa.NewIndex(indexName, nil)
	if err != nil {
		return nil, fmt.Errorf("pilosa.NewIndex: %v", err)
	}
	err = client.EnsureIndex(index)
	if err != nil {
		return nil, fmt.Errorf("client.EnsureIndex: %v", err)
	}

	// TODO should be automatic from /schema
	frames := []string{
		"lo_quantity", // these frames X each have one field, field_X
		"lo_extended_price",
		"lo_discount",
		"lo_revenue",
		"lo_supplycost",
		"lo_profit",
		"lo_revenue_computed",
		"c_city",
		"c_nation",
		"c_region",
		"s_city",
		"s_nation",
		"s_region",
		"p_mfgr",
		"p_category",
		"p_brand1",
		"lo_year",
		"lo_month",
		"lo_weeknum",
	}

	for _, frameName := range frames {
		frame, err := index.Frame(frameName, nil)
		if err != nil {
			return nil, fmt.Errorf("index.Frame %v: %v", frameName, err)
		}
		err = client.EnsureFrame(frame)
		if err != nil {
			return nil, fmt.Errorf("client.EnsureFrame %v: %v", frameName, err)
		}

		server.Frames[frameName] = frame
	}

	server.Router = router
	server.Client = client
	server.Index = index
	server.NumLineOrders = server.getLineOrderCount()
	return server, nil
}

func (s *Server) getLineOrderCount() uint64 {
	var count uint64 = 0
	for n := 0; n < 5; n++ {
		q := s.Index.Count(s.Frames["p_mfgr"].Bitmap(uint64(n)))
		response, _ := s.Client.Query(q, nil)
		count += response.Result().Count
	}
	return count
}

func (s *Server) HandleVersion(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(struct {
		DemoVersion   string `json:"demoversion"`
		PilosaVersion string `json:"pilosaversion"`
	}{
		DemoVersion:   Version,
		PilosaVersion: getPilosaVersion(),
	}); err != nil {
		log.Printf("write version response error: %s", err)
	}
}

type versionResponse struct {
	Version string `json:"version"`
}

func getPilosaVersion() string {
	resp, _ := http.Get("http://" + host + "/version")
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	version := new(versionResponse)
	json.Unmarshal(body, &version)
	return version.Version
}

func (s *Server) Serve() {
	fmt.Println("running at http://127.0.0.1:8000")
	log.Fatal(http.ListenAndServe(":8000", s.Router))
}