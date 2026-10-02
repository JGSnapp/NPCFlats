package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"backend/models"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DistrList struct {
	Districts []models.DistrictShort `json:"districts,omitempty" bson:"districts,omitempty"`
}

type FlatData struct {
	ID       string            `json:"id" bson:"id"`
	FlatID   string            `json:"flatid" bson:"flatid"`
	House    int               `json:"house" bson:"house,omitempty"`
	District int               `json:"district" bson:"district,omitempty"`
	Chair    *models.Furniture `json:"chair" bson:"chair,omitempty"`
	Table    *models.Furniture `json:"table" bson:"table,omitempty"`
	Locker   *models.Furniture `json:"locker" bson:"locker,omitempty"`
	TV       *models.Furniture `json:"tv" bson:"tv,omitempty"`
	Lamp     *models.Furniture `json:"lamp" bson:"lamp,omitempty"`
	Price    int               `json:"price" bson:"price,omitempty"`
	Men      []models.Man      `json:"men" bson:"men,omitempty"`
	Time     string            `json:"time" bson:"time"`
}

type ChatMember struct {
	Status string `json:"status"`
}

type ShopInfo struct {
	ManPrice  int                `json:"manprice"`
	FurPrice  int                `json:"furprice"`
	Furniture []models.Furniture `json:"furniture"`
}

type Sub struct {
	ImageUrl  string `json:"imageurl"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	ChannelID string `json:"channelid"`
	Sum       int    `json:"sum"`
}

type ApiResponse struct {
	Ok     bool       `json:"ok"`
	Result ChatMember `json:"result"`
}

type RequestData struct {
	ID       string `json:"id"`
	RefID    string `json:"refid"`
	District int    `json:"district"`
	House    int    `json:"house"`
	Flat     string `json:"flat"`
	FurID    int    `json:"furid"`
	RefCount int    `json:"refcount"`
}

type RequestAuction struct {
	How  int64 `json:"how"`
	Skip int64 `json:"skip"`
}

type SubData struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type DayResponse struct {
	Type      string           `json:"type"`
	Man       models.Man       `json:"man"`
	Money     int              `json:"money"`
	Furniture models.Furniture `json:"furniture"`
}

const (
	apiURL      = "https://api.telegram.org/bot%s/%s"
	checkMethod = "getChatMember"
)

var (
	backCount          int    = 0
	botToken           string = os.Getenv("BOT_TOKEN")
	dbClient           *mongo.Client
	distr_collection   *mongo.Collection
	user_collection    *mongo.Collection
	flat_collection    *mongo.Collection
	distrList          DistrList
	people             []models.Man
	channels           []Sub
	sumChance          int
	econom_furniture   []models.Furniture
	comfort_furniture  []models.Furniture
	buisness_furniture []models.Furniture
	premium_furniture  []models.Furniture
	shop               []models.Furniture
	peopleCount        int = -2
	randManCost        int = 500
	randFurCost        int = 500
	httpClient             = &http.Client{}
)

func StartMaker() {
	log.Print("StartMaker")

	backFiles, err := os.ReadDir("./flat_images")
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return
	}
	backCount = len(backFiles)

	channels_file, err := os.Open(filepath.Join("channels.txt"))
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer channels_file.Close() // Ensure file is always closed after open

	channels_scanner := bufio.NewScanner(channels_file)
	var channels_lines []string
	for channels_scanner.Scan() {
		channels_lines = append(channels_lines, channels_scanner.Text())
	}

	if channels_scanner.Err() != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	for i := 0; i+4 < len(channels_lines); i += 5 {
		sum, err := strconv.Atoi(channels_lines[i+4])
		if err != nil {
			log.Printf("Error converting allFlats: %v", err)
			continue
		}
		sub := *&Sub{
			ImageUrl:  channels_lines[i],
			Name:      channels_lines[i+1],
			URL:       channels_lines[i+2],
			ChannelID: channels_lines[i+3],
			Sum:       sum,
		}
		channels = append(channels, sub)
	}

	people_file, err := os.Open(filepath.Join("people.txt"))
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer people_file.Close() // Ensure file is always closed after open

	people_scanner := bufio.NewScanner(people_file)
	var people_lines []string
	for people_scanner.Scan() {
		people_lines = append(people_lines, people_scanner.Text())
	}

	if people_scanner.Err() != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	for i := 0; i+2 < len(people_lines); i += 3 {
		chance, err := strconv.Atoi(people_lines[i+2])
		if err != nil {
			log.Printf("Error converting allFlats: %v", err)
			continue
		}
		man := *&models.Man{
			ID:          i + 1,
			Type:        people_lines[i],
			Description: people_lines[i+1],
			Chance:      chance,
		}
		people = append(people, man)
		sumChance += man.Chance
	}

	peopleFiles, err := os.ReadDir("./people_images")
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return
	}
	peopleCount += len(peopleFiles)

	furniture_file, err := os.Open(filepath.Join("furniture.txt"))
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer furniture_file.Close() // Ensure file is always closed after open

	furniture_scanner := bufio.NewScanner(furniture_file)
	var furniture_lines []string
	for furniture_scanner.Scan() {
		furniture_lines = append(furniture_lines, furniture_scanner.Text())
	}

	if furniture_scanner.Err() != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	for i := 0; i+7 < len(furniture_lines); i += 8 {
		id, err := strconv.Atoi(furniture_lines[i])
		if err != nil {
			log.Printf("Error converting allFlats: %v", err)
			continue
		}
		quality, err := strconv.Atoi(furniture_lines[i+3])
		if err != nil {
			log.Printf("Error converting allFlats: %v", err)
			continue
		}
		price, err := strconv.Atoi(furniture_lines[i+6])
		if err != nil {
			log.Printf("Error converting allFlats: %v", err)
			continue
		}
		thing := *&models.Furniture{
			ID:          id,
			Type:        furniture_lines[i+1],
			Collection:  furniture_lines[i+2],
			Quality:     quality,
			Description: furniture_lines[i+4],
			Name:        furniture_lines[i+5],
			Price:       price,
			Skin:        furniture_lines[i+7],
		}
		log.Print(thing.Price)
		if quality == 1 {
			econom_furniture = append(econom_furniture, thing)
		} else if quality == 2 {
			comfort_furniture = append(comfort_furniture, thing)
		} else if quality == 3 {
			buisness_furniture = append(buisness_furniture, thing)
		} else {
			premium_furniture = append(premium_furniture, thing)
		}
	}

	files, err := os.ReadDir("./districts")
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return
	}

	for _, file := range files {
		log.Print(file.Name())
		if !file.IsDir() {
			f, err := os.Open(filepath.Join("./districts", file.Name()))
			if err != nil {
				log.Printf("Error opening file: %v", err)
				continue
			}
			defer f.Close() // Ensure file is always closed after open

			scanner := bufio.NewScanner(f)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			if scanner.Err() != nil {
				log.Printf("Error reading file: %v", err)
				continue
			}

			var houses []models.House
			var oldDistr models.District
			distrId, err := strconv.Atoi(lines[0]) // Check for error from Atoi
			if err != nil {
				log.Printf("Error converting distrId: %v", err)
				continue
			}

			err = distr_collection.FindOne(context.Background(), bson.M{"id": distrId}).Decode(&oldDistr)
			if err != nil {
				log.Printf("Error finding district: %v", err)
			}

			allHouses := len(oldDistr.Houses)
			for i := 3; i+5 < len(lines); i += 6 {
				houseId, err := strconv.Atoi(lines[i])
				if err != nil {
					log.Printf("Error converting houseId: %v", err)
					continue
				}
				tir, err := strconv.Atoi(lines[i+3])
				if err != nil {
					log.Printf("Error converting tir: %v", err)
					continue
				}
				price, err := strconv.Atoi(lines[i+4])
				if err != nil {
					log.Printf("Error converting price: %v", err)
					continue
				}
				allFlats, err := strconv.Atoi(lines[i+5])
				if err != nil {
					log.Printf("Error converting allFlats: %v", err)
					continue
				}

				curFlats := 0
				if i < allHouses {
					curFlats = oldDistr.Houses[i].CurFlats
				}
				house := models.House{
					ID:       houseId,
					Name:     lines[i+1],
					Tir:      tir,
					Image:    lines[i+2],
					AllFlats: allFlats,
					CurFlats: curFlats,
					Price:    price,
				}
				//log.Print("house.ID")
				//log.Print(house.ID)
				//log.Print("house.Name")
				//log.Print(house.Name)
				//log.Print("house.Tir")
				//log.Print(house.Tir)
				//log.Print("house.Image")
				//log.Print(house.Image)
				//log.Print("house.AllFlats")
				//log.Print(house.AllFlats)
				//log.Print("house.CurFlats")
				//log.Print(house.CurFlats)
				//log.Print("house.Price")
				//log.Print(house.Price)
				houses = append(houses, house)
			}
			tir, err := strconv.Atoi(lines[2])
			if err != nil {
				log.Printf("Error converting district Tir: %v", err)
				continue
			}

			distr := models.District{
				ID:     distrId,
				Name:   lines[1],
				Tir:    tir,
				Houses: houses,
			}

			distrShort := models.DistrictShort{
				ID:     distrId,
				Name:   lines[1],
				Tir:    tir,
				Houses: len(houses),
			}
			log.Print("distrShort.ID")
			log.Print(distrShort.ID)
			log.Print("distrShort.Name")
			log.Print(distrShort.Name)
			log.Print("distrShort.Tir")
			log.Print(distrShort.Tir)
			log.Print("distrShort.Houses")
			log.Print(distrShort.Houses)

			distrList.Districts = append(distrList.Districts, distrShort)

			opts := options.Update().SetUpsert(true)
			filter := bson.M{"id": distrId}
			update := bson.M{"$set": distr}

			_, err = distr_collection.UpdateOne(context.Background(), filter, update, opts)
			if err != nil {
				log.Printf("Error updating/inserting district: %v", err)
				return
			}
		}
	}
}

func AddMoney(userID string, add int) {
	log.Print("AddMoney")
	var user models.User
	err := user_collection.FindOne(context.Background(), bson.M{"id": userID}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	filter := bson.M{"id": userID}
	update := bson.M{"$inc": bson.M{"money": add}} // Используйте $inc для атомарного инкремента

	_, err = user_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
	if user.RefID != "-1" {
		var refUser models.User
		err := user_collection.FindOne(context.Background(), bson.M{"id": user.RefID}).Decode(&refUser)
		if err != nil {
			log.Printf("Error finding user: %v", err)
			return
		}
		filter := bson.M{"id": user.RefID}
		update := bson.M{"$inc": bson.M{"money": int(float64(add) * 0.1)}} // Используйте $inc для атомарного инкремента

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
		if refUser.RefID != "-1" {
			filter := bson.M{"id": refUser.RefID}
			update := bson.M{"$inc": bson.M{"money": int(float64(add) * 0.05)}} // Используйте $inc для атомарного инкремента

			_, err := user_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		}
	}
}

func EnterHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("EnterHandler")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID
	refId := requestData.RefID
	if refId == "" {
		refId = "-1"
	}
	log.Printf("refId: %v", refId)
	log.Printf("id: %v", id)

	var existingUser models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&existingUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Print("user not found")
			var firstDistrict models.District
			err := distr_collection.FindOne(context.Background(), bson.M{"id": 1}).Decode(&firstDistrict)
			firstDistrict.Houses[0].CurFlats += 1
			filter := bson.M{"id": 1}
			update := bson.M{"$set": firstDistrict}
			_, err = distr_collection.UpdateOne(context.Background(), filter, update)
			newFlat := models.Flat{
				House:     1,
				District:  1,
				OnePrice:  150000,
				Price:     0,
				Men:       make([]models.Man, 0),
				Time:      time.Now().Format(time.RFC3339),
				StartTime: time.Now().Format(time.RFC3339),
				Back:      "flat_" + fmt.Sprintf("%d", rand.Intn(backCount)+1) + ".png",
			}
			res, err := flat_collection.InsertOne(context.Background(), newFlat)
			if err != nil {
				log.Printf("error occurred while saving message: %v", err)
			}
			newFlat.ID = res.InsertedID.(primitive.ObjectID).Hex()

			filter = bson.M{"_id": res.InsertedID}
			update = bson.M{"$set": bson.M{"id": newFlat.ID}}

			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating flat: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			newMan := models.Man{
				ID:          1,
				Skin:        "Vasia.png",
				Type:        people[0].Type,
				Description: people[0].Description,
				Chance:      people[0].Chance,
			}
			newUser := models.User{
				ID:           id,
				Money:        1000000,
				Furniture:    make([]models.Furniture, 0),
				Men:          []models.Man{newMan},
				ManBound:     make([]string, 0),
				Flats:        []string{newFlat.ID},
				RefID:        refId,
				RefCount:     0,
				RefLastCheck: 0,
				Time:         time.Now().Format(time.RFC3339),
				Channels:     make([]string, 0),
				Challenge:    []bool{false, false, false, false, false, false, false, false, false, false},
			}
			res, err = user_collection.InsertOne(context.Background(), newUser)
			json.NewEncoder(w).Encode(newUser)
			if refId != "-1" {
				var refUser models.User
				err := user_collection.FindOne(context.Background(), bson.M{"id": refId}).Decode(&refUser)
				if err != nil {
					log.Printf("Error finding user: %v", err)
					return
				}
				filter := bson.M{"id": refId}
				update := bson.M{"$inc": bson.M{"refcount": 1}}

				_, err = user_collection.UpdateOne(context.Background(), filter, update)
				if err != nil {
					log.Printf("error occurred while updating user: %v", err)
					return
				}
				if refUser.RefID != "-1" {
					var refRefUser models.User
					err := user_collection.FindOne(context.Background(), bson.M{"id": refUser.RefID}).Decode(&refRefUser)
					if err != nil {
						log.Printf("Error finding user: %v", err)
						return
					}
					filter := bson.M{"id": refUser.RefID}
					update := bson.M{"$inc": bson.M{"refcount": 1}}

					_, err = user_collection.UpdateOne(context.Background(), filter, update)
					if err != nil {
						log.Printf("error occurred while updating user: %v", err)
						return
					}
				}
			}
		}
	} else {
		log.Print("user exist")
		existingUser.Time = time.Now().Format(time.RFC3339)
		filter := bson.M{"id": id}
		update := bson.M{"$set": existingUser}

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating flat: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(existingUser)
	}
}

func GetDistricts(w http.ResponseWriter, r *http.Request) {
	log.Print("GetDistricts")
	json.NewEncoder(w).Encode(distrList)
}

func GetHouses(w http.ResponseWriter, r *http.Request) {
	log.Print("GetHouses")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		log.Printf("Error finding district: %v", err)
		return
	}
	district := requestData.District
	var distr models.District
	err = distr_collection.FindOne(context.Background(), bson.M{"id": district}).Decode(&distr)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	} else {
		log.Print(distr.ID)
		log.Print(distr.Name)
		log.Print(distr.Houses[0].Name)
		json.NewEncoder(w).Encode(distr)
	}
}

func BuyFlat(w http.ResponseWriter, r *http.Request) {
	log.Print("BuyFlat")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID
	district := requestData.District
	house := requestData.House
	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	var distr models.District
	err = distr_collection.FindOne(context.Background(), bson.M{"id": district}).Decode(&distr)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	payHouse := distr.Houses[house-1]
	if (payHouse.CurFlats < payHouse.AllFlats || payHouse.AllFlats == -1) && user.Money >= payHouse.Price {
		user.Money -= payHouse.Price
		payHouse.CurFlats += 1
		newFlat := models.Flat{
			House:     house,
			District:  district,
			OnePrice:  payHouse.Price,
			Price:     0,
			Men:       make([]models.Man, 0),
			Time:      time.Now().Format(time.RFC3339),
			StartTime: time.Now().Format(time.RFC3339),
			Auction:   false,
			Back:      "flat_" + fmt.Sprintf("%d", rand.Intn(backCount)+1) + ".png",
		}
		res, err := flat_collection.InsertOne(context.Background(), newFlat)
		if err != nil {
			log.Printf("error occurred while saving message: %v", err)
		}
		newFlat.ID = res.InsertedID.(primitive.ObjectID).Hex()

		filter := bson.M{"_id": res.InsertedID}
		update := bson.M{"$set": bson.M{"id": newFlat.ID}}

		_, err = flat_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating flat: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		user.Flats = append(user.Flats, newFlat.ID)
		distr.Houses[house-1] = payHouse
		filter = bson.M{"id": district}
		update = bson.M{"$set": bson.M{"houses": distr.Houses}}

		_, err = distr_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating flat: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		filter = bson.M{"id": id}
		update = bson.M{"$set": user}

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating flat: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(newFlat)
	}
}

func TakeMoney(w http.ResponseWriter, r *http.Request) {
	log.Print("TakeMoney")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID
	flat := requestData.Flat

	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var moneyFlat models.Flat
	err = flat_collection.FindOne(context.Background(), bson.M{"id": flat}).Decode(&moneyFlat)
	if err != nil {
		log.Printf("Error finding flat: %v", err)
		http.Error(w, "Flat not found", http.StatusNotFound)
		return
	}
	parsedTime, err := time.Parse(time.RFC3339, moneyFlat.Time)
	if err != nil {
		fmt.Println("Ошибка при разборе времени:", err)
		return
	}
	if time.Since(parsedTime) > 1*time.Minute {

		filter := bson.M{"id": flat}
		update := bson.M{"$set": bson.M{"time": time.Now().Format(time.RFC3339)}}
		_, err = flat_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("Error updating flat: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		AddMoney(id, moneyFlat.Price)
	}
}

func GetFlat(w http.ResponseWriter, r *http.Request) {
	log.Print("GetFlat")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	flat := requestData.Flat
	var moneyFlat models.Flat
	err = flat_collection.FindOne(context.Background(), bson.M{"id": flat}).Decode(&moneyFlat)
	if err != nil {
		log.Printf("Error finding flat: %v", err)
		http.Error(w, "Flat not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(moneyFlat)
}

func ReMarketScheduler() {
	ReMarket()
	ticker := time.NewTicker(6 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ReMarket()
		}
	}
}

func ReMarket() {
	shop = []models.Furniture{}

	// Добавление одного случайного элемента из buisness_furniture
	if len(buisness_furniture) > 0 {
		randomIndex := rand.Intn(len(buisness_furniture))
		shop = append(shop, buisness_furniture[randomIndex])
	}

	u := 0
	y := 0
	for i := 0; i < 3; i++ {
		if len(comfort_furniture) > 0 {
			randomIndex := rand.Intn(len(comfort_furniture))
			for u == randomIndex || y == randomIndex {
				randomIndex = rand.Intn(len(comfort_furniture))
			}
			y = u
			u = randomIndex
			shop = append(shop, comfort_furniture[randomIndex])
		}
	}
}

func GetMarket(w http.ResponseWriter, r *http.Request) {
	log.Print("GetMarket")
	shopInfo := ShopInfo{
		Furniture: shop,
		FurPrice:  randFurCost,
		ManPrice:  randManCost,
	}
	log.Print(shop)
	log.Print(randFurCost)
	log.Print(randManCost)
	json.NewEncoder(w).Encode(shopInfo)
}

func BuyGuy(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID

	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	if user.Money >= randManCost {
		randMan := GetRandomGuy()
		if len(user.ManBound) < 5 {
			for {
				exist := false
				for i := 0; i < len(user.ManBound); i++ {
					if user.ManBound[i] == randMan.Type || randMan.Type == "Беспринципный" {
						randMan = GetRandomGuy()
						exist = true
					}
				}
				if !exist {
					break
				}
			}
			user.ManBound = append(user.ManBound, randMan.Type)
		}

		randomIndex := rand.Intn(peopleCount) + 1
		randMan.Skin = fmt.Sprintf("person_%d.png", randomIndex)
		user.Men = append(user.Men, randMan)
		user.Money -= randManCost

		filter := bson.M{"id": id}
		update := bson.M{"$set": user}

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}

		json.NewEncoder(w).Encode(randMan)
	}
}

func GetRandomGuy() models.Man {
	randValue := rand.Intn(sumChance)
	for _, man := range people {
		randValue -= man.Chance
		if randValue < 0 {
			return man
		}
	}
	return models.Man{}
}

func BuyFur(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID

	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	if user.Money >= randFurCost {
		randomProduct := getRandomProduct()
		user.Furniture = append(user.Furniture, randomProduct)
		user.Money -= randFurCost
		filter := bson.M{"id": id}
		update := bson.M{"$set": user}

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
		json.NewEncoder(w).Encode(randomProduct)
	}
}

func getRandomProduct() models.Furniture {
	randValue := rand.Intn(100) // Генерация случайного числа от 0 до 99

	switch {
	case randValue < 60:
		return comfort_furniture[rand.Intn(len(comfort_furniture))]
	case randValue < 90:
		return premium_furniture[rand.Intn(len(premium_furniture))]
	default:
		return buisness_furniture[rand.Intn(len(buisness_furniture))]
	}
}

func GetProduct(id int) models.Furniture {
	for _, product := range shop {
		log.Print("Shop_try")
		log.Print(product.ID)
		if product.ID == id {
			log.Print("YES!!!")
			log.Print(product.ID)
			return product
		}
	}
	return models.Furniture{}
}

func BuyIndex(w http.ResponseWriter, r *http.Request) {
	log.Print("BuyIndex")
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	furID := requestData.FurID
	id := requestData.ID
	product := GetProduct(furID)
	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	if product.Price <= user.Money {
		user.Furniture = append(user.Furniture, product)
		user.Money -= product.Price

		log.Print(product.Price)
		log.Print(user.Money)
		log.Print(user.Furniture[len(user.Furniture)-1].Name)

		filter := bson.M{"id": id}
		update := bson.M{"$set": user}

		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
	}
}

func UpdateFlat(w http.ResponseWriter, r *http.Request) {
	log.Print("UpdateFlat")
	var flatData FlatData
	err := json.NewDecoder(r.Body).Decode(&flatData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := flatData.ID
	flatID := flatData.FlatID
	chair := flatData.Chair
	table := flatData.Table
	lamp := flatData.Lamp
	locker := flatData.Locker
	tv := flatData.TV
	men := flatData.Men
	log.Print(id)
	log.Print(flatID)
	log.Print(chair)
	log.Print(table)
	log.Print(lamp)
	log.Print(locker)
	log.Print(tv)
	log.Print(men)

	newFurnitures := []*models.Furniture{chair, table, lamp, locker, tv}
	maxItemsPerLevel := make(map[int]map[string]int)

	// Подсчёт предметов по коллекциям и качеству
	for _, f := range newFurnitures {
		if f != nil {
			if maxItemsPerLevel[f.Quality] == nil {
				maxItemsPerLevel[f.Quality] = make(map[string]int)
			}
			maxItemsPerLevel[f.Quality][f.Collection]++
		}
	}

	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}

	for i := 0; i < len(user.Men); i++ {
		log.Print("mmm")
		log.Print(user.Men[i])
	}

	var existFlat models.Flat
	err = flat_collection.FindOne(context.Background(), bson.M{"id": flatID}).Decode(&existFlat)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	var existDistr models.District
	err = distr_collection.FindOne(context.Background(), bson.M{"id": existFlat.District}).Decode(&existDistr)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	house := existDistr.Houses[existFlat.House-1]

	if chair != existFlat.Chair {
		if chair == nil {
			user.Furniture = append(user.Furniture, *existFlat.Chair)
			filter := bson.M{"id": flatID}
			update := bson.M{
				"$unset": bson.M{
					"chair": "",
				},
			}
			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		} else {
			if existFlat.Chair != nil {
				user.Furniture = append(user.Furniture, *existFlat.Chair)
			}
			user.Furniture = removeFurByID(user.Furniture, chair.ID)
		}
		existFlat.Chair = chair
	}
	if table != existFlat.Table {
		if table == nil {
			user.Furniture = append(user.Furniture, *existFlat.Table)
			filter := bson.M{"id": flatID}
			update := bson.M{
				"$unset": bson.M{
					"table": "",
				},
			}
			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		} else {
			if existFlat.Table != nil {
				user.Furniture = append(user.Furniture, *existFlat.Table)
			}
			user.Furniture = removeFurByID(user.Furniture, table.ID)
		}
		existFlat.Table = table
	}
	if lamp != existFlat.Lamp {
		if lamp == nil {
			user.Furniture = append(user.Furniture, *existFlat.Lamp)
			filter := bson.M{"id": flatID}
			update := bson.M{
				"$unset": bson.M{
					"lamp": "",
				},
			}
			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		} else {
			if existFlat.Lamp != nil {
				user.Furniture = append(user.Furniture, *existFlat.Lamp)
			}
			user.Furniture = removeFurByID(user.Furniture, lamp.ID)
		}
		existFlat.Lamp = lamp
	}
	if locker != existFlat.Locker {
		if locker == nil {
			user.Furniture = append(user.Furniture, *existFlat.Locker)
			filter := bson.M{"id": flatID}
			update := bson.M{
				"$unset": bson.M{
					"locker": "",
				},
			}
			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		} else {
			if existFlat.Locker != nil {
				user.Furniture = append(user.Furniture, *existFlat.Locker)
			}
			user.Furniture = removeFurByID(user.Furniture, locker.ID)
		}
		existFlat.Locker = locker
	}
	if tv != existFlat.TV {
		if tv == nil {
			user.Furniture = append(user.Furniture, *existFlat.TV)
			filter := bson.M{"id": flatID}
			update := bson.M{
				"$unset": bson.M{
					"tv": "",
				},
			}
			_, err = flat_collection.UpdateOne(context.Background(), filter, update)
			if err != nil {
				log.Printf("error occurred while updating user: %v", err)
				return
			}
		} else {
			if existFlat.TV != nil {
				user.Furniture = append(user.Furniture, *existFlat.TV)
			}
			user.Furniture = removeFurByID(user.Furniture, tv.ID)
		}
		existFlat.TV = tv
	}

	newPrice := float64(existFlat.OnePrice) * 0.06
	switch existDistr.Tir {
	case 3:
		switch house.Tir {
		case 4:
			if checkLevel(maxItemsPerLevel, 4, 5) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		case 3:
			if checkLevel(maxItemsPerLevel, 3, 5) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		case 2:
			if checkLevel(maxItemsPerLevel, 2, 5) && checkLevel(maxItemsPerLevel, 3, 2) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		default:
			fmt.Println("111")
		}
	case 2:
		switch house.Tir {
		case 3:
			if checkLevel(maxItemsPerLevel, 2, 5) && checkLevel(maxItemsPerLevel, 3, 2) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		case 2:
			if checkLevel(maxItemsPerLevel, 2, 5) && checkLevel(maxItemsPerLevel, 3, 1) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		case 1:
			if checkLevel(maxItemsPerLevel, 2, 5) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		default:
			fmt.Println("111")
		}
	case 1:
		switch house.Tir {
		case 2:
			if checkLevel(maxItemsPerLevel, 2, 5) && checkLevel(maxItemsPerLevel, 3, 1) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		case 1:
			if checkLevel(maxItemsPerLevel, 2, 4) || (checkLevel(maxItemsPerLevel, 2, 2) && checkLevel(maxItemsPerLevel, 3, 1)) {
				newPrice *= 1.5
			} else {
				log.Print("house_no")
			}
		default:
			fmt.Println("111")
		}
	default:
		fmt.Println("111")
	}
	happyMeal := 0
	user.Men = append(user.Men, existFlat.Men...)
	for _, man := range men {
		switch man.Type {
		case "Одиночка":
			if len(men) == 1 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Душа компании":
			if len(men) == 3 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Вася":
			if checkLevel(maxItemsPerLevel, 4, 5) && house.Tir == 4 && existDistr.ID == 4 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Городской":
			if existDistr.Tir > 1 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Скромняга":
			if existDistr.Tir < 3 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Деревенщина":
			if existDistr.Tir == 1 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Мажор":
			if existDistr.Tir == 3 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Экономист":
			if house.Tir == 1 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Бизнесмен":
			if house.Tir == 3 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Скряга":
			if house.Tir < 3 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Транжира":
			if house.Tir > 4 {
				happyMeal += 1
			} else {
				log.Print("no")
			}
		case "Беспринципные":
			happyMeal += 1
		}
		user.Men = removeFirstMatch(user.Men, man.Type, man.Skin)
	}
	existFlat.Men = flatData.Men
	switch happyMeal {
	case 0:
		if len(flatData.Men) == 0 {
			newPrice = 0
		}
	case 1:
		newPrice *= 1.5
	case 2:
		newPrice *= 2.25
	case 3:
		newPrice *= 3
	}
	existFlat.Price = int(newPrice)

	if existFlat.Price == 0 {
		filter := bson.M{"id": flatID}
		update := bson.M{"$set": bson.M{"price": 0}}

		_, err = flat_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
	}

	log.Print("price")
	log.Print(existFlat.Price)

	filter := bson.M{"id": id}
	update := bson.M{"$set": user}

	_, err = user_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
	filter = bson.M{"id": flatID}
	update = bson.M{"$set": existFlat}

	_, err = flat_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
	if len(men) == 0 {
		log.Print("-men")
		filter := bson.M{"id": flatID}
		update := bson.M{
			"$set": bson.M{
				"men": bson.A{},
			},
		}
		_, err := flat_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Fatalf("Failed to clear array: %v", err)
		}
	}
	if len(user.Furniture) == 0 {
		log.Print("-fur user")
		filter := bson.M{"id": id}
		update := bson.M{
			"$set": bson.M{
				"furniture": bson.A{},
			},
		}
		_, err := user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Fatalf("Failed to clear array: %v", err)
		}
	}
	if len(user.Men) == 0 {
		log.Print("-men user")
		filter := bson.M{"id": id}
		update := bson.M{
			"$set": bson.M{
				"men": bson.A{},
			},
		}
		_, err := user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Fatalf("Failed to clear array: %v", err)
		}
	}
	log.Print("user.Furniture")
	json.NewEncoder(w).Encode(existFlat)
}

func removeFirstMatch(men []models.Man, targetType, targetSkin string) []models.Man {
	log.Print("targetType")
	log.Print(targetType)
	log.Print("targetSkin")
	log.Print(targetSkin)
	for i, man := range men {
		log.Print("targetType__n")
		log.Print(man.Type)
		log.Print("targetSkin__n")
		log.Print(man.Skin)
		if man.Type == targetType && man.Skin == targetSkin {
			log.Print("!!!")
			return append(men[:i], men[i+1:]...)
		}
	}
	return men
}

func checkLevel(maxItemsPerLevel map[int]map[string]int, level int, expected int) bool {
	actualCount := 0
	for lvl, collections := range maxItemsPerLevel {
		if lvl >= level {
			for _, count := range collections {
				if count > actualCount {
					actualCount = count
				}
			}
		}
	}
	return actualCount >= expected
}

func removeFurByID(furniture []models.Furniture, id int) []models.Furniture {
	for i, thing := range furniture {
		if thing.ID == id {
			return append(furniture[:i], furniture[i+1:]...)
		}
	}
	return furniture
}

func SellFlat(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	flatID := requestData.Flat
	id := requestData.ID
	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	var soldflat models.Flat
	err = flat_collection.FindOne(context.Background(), bson.M{"id": flatID}).Decode(&soldflat)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	parsedTime, err := time.Parse(time.RFC3339, soldflat.StartTime)
	if err != nil {
		fmt.Println("Ошибка при разборе времени:", err)
		return
	}
	days := int(time.Since(parsedTime).Hours() / 24)
	discountFactor := math.Pow(0.98, float64(days))
	price := int(float64(soldflat.OnePrice) * discountFactor)
	user.Money += price
	soldflat.Price = price
	soldflat.Auction = true
	for i, flat := range user.Flats {
		if flat == flatID {
			user.Flats = append(user.Flats[:i], user.Flats[i+1:]...)
		}
	}
	filter := bson.M{"id": id}
	update := bson.M{"$set": user}

	_, err = user_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
	filter = bson.M{"id": flatID}
	update = bson.M{"$set": soldflat}

	_, err = flat_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
}

func GetAuction(w http.ResponseWriter, r *http.Request) {
	var requestAuction RequestAuction
	err := json.NewDecoder(r.Body).Decode(&requestAuction)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	how := requestAuction.How
	skip := requestAuction.Skip
	var results []models.Flat
	findOptions := options.Find()
	findOptions.SetLimit(how)
	findOptions.SetSkip(skip)
	filter := bson.M{"auction": true}
	cur, err := flat_collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(context.TODO())
	for cur.Next(context.TODO()) {
		var elem models.Flat
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, elem)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(w).Encode(results)
}

func BuyAuction(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	flatID := requestData.Flat
	id := requestData.ID
	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	var soldflat models.Flat
	err = flat_collection.FindOne(context.Background(), bson.M{"id": flatID}).Decode(&soldflat)
	if err != nil {
		log.Printf("Error finding district: %v", err)
	}
	if user.Money >= soldflat.Price {
		user.Money -= soldflat.Price
		user.Flats = append(user.Flats, soldflat.ID)
		soldflat.Auction = false
		filter := bson.M{"id": id}
		update := bson.M{"$set": user}
		_, err = user_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
		filter = bson.M{"id": flatID}
		update = bson.M{"$set": soldflat}
		_, err = flat_collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("error occurred while updating user: %v", err)
			return
		}
	}
}

func checkSubscription(w http.ResponseWriter, r *http.Request) {
	log.Print("checkSubscription")
	var subData SubData
	if err := json.NewDecoder(r.Body).Decode(&subData); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Retrieve the user from the database
	var user models.User
	err := user_collection.FindOne(context.Background(), bson.M{"id": subData.ID}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var channel *Sub // указатель на структуру Person
	for _, channel1 := range channels {
		if channel1.Name == subData.Name {
			channel = &channel1 // сохраняем адрес найденной структуры
			break               // прерываем цикл после нахождения первого совпадения
		}
	}
	log.Print("name")
	log.Print(channel.Name)

	chatId := channel.ChannelID

	userId, err := strconv.Atoi(subData.ID)
	if err != nil {
		log.Printf("Error converting allFlats: %v", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=%s&user_id=%d", botToken, chatId, userId)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Проверяем, что запрос был успешным
	if result["ok"].(bool) {
		chatMember := result["result"].(map[string]interface{})
		status := chatMember["status"].(string)

		// Проверяем статус пользователя
		if status == "member" || status == "administrator" || status == "creator" {
			log.Print("Yesssss")
			yet := true
			for _, channel := range user.Channels {
				if channel == chatId {
					yet = false
					break
				}
			}
			if yet {
				var win DayResponse
				user.Channels = append(user.Channels, chatId)
				if channel.Sum != 0 {
					log.Print("money")
					user.Money += channel.Sum
					win.Type = "money"
					win.Money = channel.Sum
				} else {
					log.Print("man")
					win.Type = "man"
					randomIndex := rand.Intn(peopleCount) + 1
					newMan := models.Man{
						ID:          3,
						Skin:        fmt.Sprintf("person_%d.png", randomIndex),
						Type:        people[2].Type,
						Description: people[2].Description,
						Chance:      people[2].Chance,
					}
					user.Men = append(user.Men, newMan)
					win.Man = newMan
				}
				filter := bson.M{"id": subData.ID}
				update := bson.M{"$set": user}
				if _, err := user_collection.UpdateOne(context.Background(), filter, update); err != nil {
					log.Printf("Error occurred while updating user: %v", err)
				}
				json.NewEncoder(w).Encode(win)
			}
		} else {
			log.Print("NOoOoOooO")
		}
	} else {
		fmt.Println("Ошибка при получении информации о пользователе:", result["description"].(string))
	}
}

func CheckRef(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID
	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		return
	}
	count := user.RefCount
	yet := user.RefLastCheck
	user.RefLastCheck = count
	addGuys := make([]models.Man, 0)
	if count >= 1 && yet < 1 {
		randomIndex := rand.Intn(peopleCount) + 1
		newMan := models.Man{
			ID:          4,
			Skin:        fmt.Sprintf("person_%d.png", randomIndex),
			Type:        people[3].Type,
			Description: people[3].Description,
			Chance:      people[3].Chance,
		}
		addGuys = append(addGuys, newMan)
	}
	if count >= 3 && yet < 3 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 8 && yet < 8 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 15 && yet < 15 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 25 && yet < 25 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 50 && yet < 50 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 80 && yet < 80 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 125 && yet < 125 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 175 && yet < 175 {
		addGuys = append(addGuys, GuyForRefs())
	}
	if count >= 250 && yet < 250 {
		count = int((count - 250) / 100)
		for i := 0; i < count; i++ {
			addGuys = append(addGuys, GuyForRefs())
		}
	}
	user.Men = append(user.Men, addGuys...)
	filter := bson.M{"id": id}
	update := bson.M{"$set": user}
	_, err = user_collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("error occurred while updating user: %v", err)
		return
	}
	json.NewEncoder(w).Encode(addGuys)
}

func GuyForRefs() models.Man {
	randomNumber := rand.Intn(100)
	if randomNumber < 27 {
		randomIndex := rand.Intn(peopleCount) + 1
		newMan := models.Man{
			ID:          4,
			Skin:        fmt.Sprintf("person_%d.png", randomIndex),
			Type:        people[3].Type,
			Description: people[3].Description,
			Chance:      people[3].Chance,
		}
		return newMan
	} else {
		return GetRandomGuy()
	}
}

func GetSubscription(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(channels)
}

func CheckDays(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	id := requestData.ID

	var user models.User
	err = user_collection.FindOne(context.Background(), bson.M{"id": id}).Decode(&user)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	parsedDate, err := time.Parse(time.RFC3339, user.Time)
	if err != nil {
		fmt.Println("Ошибка при парсинге даты:", err)
		return
	}

	// Вычисление количества дней, прошедших с этой даты
	duration := time.Since(parsedDate)
	days := int(duration.Hours() / 24)
	var win DayResponse
	if days < 7 {
		user.Challenge[days] = true
		switch days {
		case 0:
			win.Type = "money"
			user.Money += 25000
			win.Money = 25000
		case 1:
			win.Type = "money"
			user.Money += 50000
			win.Money = 50000
		case 2:
			win.Type = "furniture"
			user.Furniture = append(user.Furniture, comfort_furniture[2])
			win.Furniture = comfort_furniture[2]
		case 3:
			win.Type = "money"
			user.Money += 200000
			win.Money = 200000
		case 4:
			win.Type = "furniture"
			randomIndex := rand.Intn(len(comfort_furniture))
			randFur := comfort_furniture[randomIndex]
			user.Furniture = append(user.Furniture, randFur)
			win.Furniture = randFur
		case 5:
			win.Type = "furniture"
			user.Furniture = append(user.Furniture, buisness_furniture[4])
			win.Furniture = buisness_furniture[4]
		case 6:
			win.Type = "man"
			newMan := models.Man{
				ID:          2,
				Skin:        "Larry.png",
				Type:        people[1].Type,
				Description: people[1].Description,
				Chance:      people[1].Chance,
			}
			user.Men = append(user.Men, newMan)
		}
	} else {
		win.Type = "no"
	}
	json.NewEncoder(w).Encode(win)
}

func main() {
	log.Print("main")
	rand.Seed(time.Now().UnixNano())
	clientOptions := options.Client().ApplyURI("mongodb://root:example@mongo:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		fmt.Println("333Error connecting to MongoDB:", err)
		return
	}
	dbClient = client
	distr_collection = client.Database("flatholderdb").Collection("districts")
	user_collection = client.Database("flatholderdb").Collection("users")
	flat_collection = client.Database("flatholderdb").Collection("flats")

	StartMaker()
	go ReMarketScheduler()

	r := mux.NewRouter()
	r.HandleFunc("/enter", EnterHandler).Methods("POST")
	r.HandleFunc("/districts", GetDistricts).Methods("POST")
	r.HandleFunc("/houses", GetHouses).Methods("POST")
	r.HandleFunc("/flat", GetFlat).Methods("POST")
	r.HandleFunc("/buyflat", BuyFlat).Methods("POST")
	r.HandleFunc("/takemoney", TakeMoney).Methods("POST")
	r.HandleFunc("/getmarket", GetMarket).Methods("POST")
	r.HandleFunc("/buyguy", BuyGuy).Methods("POST")
	r.HandleFunc("/buyfur", BuyFur).Methods("POST")
	r.HandleFunc("/buyindex", BuyIndex).Methods("POST")
	r.HandleFunc("/updateflat", UpdateFlat).Methods("POST")
	r.HandleFunc("/sellflat", SellFlat).Methods("POST")
	r.HandleFunc("/getauction", GetAuction).Methods("POST")
	r.HandleFunc("/buyauction", BuyAuction).Methods("POST")
	r.HandleFunc("/checksubscription", checkSubscription).Methods("POST")
	r.HandleFunc("/checkref", CheckRef).Methods("POST")
	r.HandleFunc("/getsubscription", GetSubscription).Methods("POST")
	r.HandleFunc("/checkdays", CheckDays).Methods("POST")
	currentDir, _ := os.Getwd()

	fs1 := http.FileServer(http.Dir(filepath.Join(currentDir, "districts_images")))
	fs2 := http.FileServer(http.Dir(filepath.Join(currentDir, "houses_images")))
	fs3 := http.FileServer(http.Dir(filepath.Join(currentDir, "flat_images")))
	fs4 := http.FileServer(http.Dir(filepath.Join(currentDir, "furniture_images")))
	fs5 := http.FileServer(http.Dir(filepath.Join(currentDir, "people_images")))
	fs6 := http.FileServer(http.Dir(filepath.Join(currentDir, "interface_images")))
	fs7 := http.FileServer(http.Dir(filepath.Join(currentDir, "channels_images")))

	r.PathPrefix("/districts_images/").Handler(http.StripPrefix("/districts_images/", fs1))
	r.PathPrefix("/houses_images/").Handler(http.StripPrefix("/houses_images/", fs2))
	r.PathPrefix("/flat_images/").Handler(http.StripPrefix("/flat_images/", fs3))
	r.PathPrefix("/furniture_images/").Handler(http.StripPrefix("/furniture_images/", fs4))
	r.PathPrefix("/people_images/").Handler(http.StripPrefix("/people_images/", fs5))
	r.PathPrefix("/interface_images/").Handler(http.StripPrefix("/interface_images/", fs6))
	r.PathPrefix("/channels_images/").Handler(http.StripPrefix("/channels_images/", fs7))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})

	handler := c.Handler(r)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
