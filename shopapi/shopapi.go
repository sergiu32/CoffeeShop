package shopapi

import (
	"CoffeeShop/coffeedb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type CoffeeQuota struct {
	Type      coffeedb.CoffeeType
	Amount    uint32
	TimeFrame int64
}

type CoffeeQuotaPerMembership struct {
	Membership coffeedb.MembershipType
	Quota      []CoffeeQuota
}

type CoffeeLimitExceed struct {
	Type         coffeedb.CoffeeType
	AmountBought uint32
	AvailableIn  int64
}

type UserRegister struct {
	UserId     string                  `json:"user_id"`
	Membership coffeedb.MembershipType `json:"membership"`
}

type CoffeeBuyInfo struct {
	UserId string              `json:"user_id"`
	Coffee coffeedb.CoffeeType `json:"coffee_type"`
}

func (cqm *CoffeeQuotaPerMembership) PrintConfig() {
	fmt.Printf("Membership \"%s\"\n", cqm.Membership.String())
	for _, q := range cqm.Quota {
		fmt.Printf("%d %s in Last %s\n", q.Amount, q.Type.String(), time.Duration(q.TimeFrame).String())
	}
	fmt.Println()
}

var coffeeConfig map[coffeedb.MembershipType]CoffeeQuotaPerMembership

var db *coffeedb.CoffeeDb

func coffeeQuotaConfig(coffee coffeedb.CoffeeType, membership coffeedb.MembershipType) (*CoffeeQuota, error) {
	if cq, ok := coffeeConfig[membership]; ok {
		for _, cQuotaItem := range cq.Quota {
			if cQuotaItem.Type == coffee {
				return &cQuotaItem, nil
			}
		}
		return nil, errors.New("invalid coffee type or configuration missing")
	}
	return nil, errors.New("invalid Membership")
}

func buyCoffee(userId string, coffee coffeedb.CoffeeType) (*CoffeeLimitExceed, error) {
	qs := db.GetUserData(userId)
	if qs == nil {
		return nil, errors.New("user not found " + userId)
	}
	cQuotaConfig, err := coffeeQuotaConfig(coffee, qs.Membership)
	if err != nil {
		return nil, err
	}

	timeNowSeconds := time.Now().Unix()
	if userCoffeeQuota, ok := qs.QuotaState[coffee]; ok {
		//user has bought some coffee already
		quotaInSeconds := int64(time.Duration(cQuotaConfig.TimeFrame).Seconds())
		timeDiff := timeNowSeconds - userCoffeeQuota.StartBoughtTime

		if timeDiff < quotaInSeconds {
			if userCoffeeQuota.AmountBought >= cQuotaConfig.Amount {
				//return quota limit exceeded
				return &CoffeeLimitExceed{Type: coffee, AmountBought: userCoffeeQuota.AmountBought, AvailableIn: quotaInSeconds - timeDiff}, nil
			}
			return nil, db.SetQuotaState(userId, coffee, &coffeedb.UserCoffeeQuota{AmountBought: userCoffeeQuota.AmountBought + 1, StartBoughtTime: userCoffeeQuota.StartBoughtTime})
		}
		//time has passed quota reset user amount and time
		return nil, db.SetQuotaState(userId, coffee, &coffeedb.UserCoffeeQuota{AmountBought: 1, StartBoughtTime: timeNowSeconds})
	}
	//user is buying coffee for the first time
	return nil, db.SetQuotaState(userId, coffee, &coffeedb.UserCoffeeQuota{AmountBought: 1, StartBoughtTime: timeNowSeconds})
}

func apiRegisterUser(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(writer, "Method is not supported.", http.StatusNotFound)
		return
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "could not read body", http.StatusBadRequest)
	}
	var userReg UserRegister
	err = json.Unmarshal(body, &userReg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if len(userReg.UserId) == 0 {
		http.Error(writer, "empty user id", http.StatusBadRequest)
		return
	}
	registeredUser := db.GetUserData(userReg.UserId)
	if registeredUser != nil {
		http.Error(writer, "this user is registered already", http.StatusBadRequest)
		return
	}
	err = userReg.Membership.IsValid()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	err = db.RegisterUser(userReg.UserId, userReg.Membership)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func apiBuyCoffee(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(writer, "Method is not supported.", http.StatusNotFound)
		return
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "could not read body", http.StatusBadRequest)
	}
	var cInfo CoffeeBuyInfo
	err = json.Unmarshal(body, &cInfo)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if len(cInfo.UserId) == 0 {
		http.Error(writer, "empty user id", http.StatusBadRequest)
		return
	}
	limit, err := buyCoffee(cInfo.UserId, cInfo.Coffee)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if limit != nil {
		http.Error(
			writer,
			fmt.Sprintf("User %s limit exceeded, %s bought: %d available in %s\n",
				cInfo.UserId,
				limit.Type.String(),
				limit.AmountBought,
				time.Duration(limit.AvailableIn*int64(time.Second)).String()),
			http.StatusTooManyRequests)
		return
	}
	writer.WriteHeader(http.StatusOK)
}

func clearDb() {
	db.ClearDb()
}

// InitWithConfig initialize a custom config
// it will overwrite default one if exists
func InitWithConfig(config map[coffeedb.MembershipType]CoffeeQuotaPerMembership) {
	log.Println("Initializing config...")
	coffeeConfig = config
	for _, value := range coffeeConfig {
		value.PrintConfig()
	}
}

// InitDefaultConfig initialize a default config
// something like:
// Basic:
// 1 Espresso in last 24 hours
// 2 Americano in last 24 hours
// 3 Cappuccino in last 24 hours
//
// Membership "Coffeelover":
// 5 Espresso in last 24 hours
// 5 Americano in last 24 hours
// 5 Cappuccino in last 24 hours
//
// Membership "Espresso Maniac"
// 5 Espresso in last 60 minutes
// Cappuccino/Americano same as "Basic"
func InitDefaultConfig() {
	coffeeConfig = make(map[coffeedb.MembershipType]CoffeeQuotaPerMembership)

	//basic membership config
	basicEspressoCoffeeQuota := CoffeeQuota{Type: coffeedb.Espresso, Amount: 1, TimeFrame: int64(time.Hour * 24)}
	basicAmericanoCoffeeQuota := CoffeeQuota{Type: coffeedb.Americano, Amount: 2, TimeFrame: int64(time.Hour * 24)}
	basicCappuccinoCoffeeQuota := CoffeeQuota{Type: coffeedb.Cappuccino, Amount: 3, TimeFrame: int64(time.Hour * 24)}
	basicQuotas := []CoffeeQuota{basicEspressoCoffeeQuota, basicAmericanoCoffeeQuota, basicCappuccinoCoffeeQuota}
	basicCoffeeConfig := CoffeeQuotaPerMembership{Membership: coffeedb.Basic, Quota: basicQuotas}
	coffeeConfig[coffeedb.Basic] = basicCoffeeConfig
	basicCoffeeConfig.PrintConfig()

	//CoffeeLover membership config
	coffeeLoverEspressoCoffeeQuota := CoffeeQuota{Type: coffeedb.Espresso, Amount: 5, TimeFrame: int64(time.Hour * 24)}
	coffeeLoverAmericanoCoffeeQuota := CoffeeQuota{Type: coffeedb.Americano, Amount: 5, TimeFrame: int64(time.Hour * 24)}
	coffeeLoverCappuccinoCoffeeQuota := CoffeeQuota{Type: coffeedb.Cappuccino, Amount: 5, TimeFrame: int64(time.Hour * 24)}
	coffeeLoverQuotas := []CoffeeQuota{coffeeLoverEspressoCoffeeQuota, coffeeLoverAmericanoCoffeeQuota, coffeeLoverCappuccinoCoffeeQuota}
	coffeeLoverCoffeeConfig := CoffeeQuotaPerMembership{Membership: coffeedb.CoffeeLover, Quota: coffeeLoverQuotas}
	coffeeConfig[coffeedb.CoffeeLover] = coffeeLoverCoffeeConfig
	coffeeLoverCoffeeConfig.PrintConfig()

	//Espresso Maniac membership config
	espressoManiacEspressoCoffeeQuota := CoffeeQuota{Type: coffeedb.Espresso, Amount: 5, TimeFrame: int64(time.Hour)}
	espressoManiacAmericanoCoffeeQuota := CoffeeQuota{Type: coffeedb.Americano, Amount: 2, TimeFrame: int64(time.Hour * 24)}
	espressoManiacCappuccinoCoffeeQuota := CoffeeQuota{Type: coffeedb.Cappuccino, Amount: 3, TimeFrame: int64(time.Hour * 24)}
	espressoManiacQuotas := []CoffeeQuota{espressoManiacEspressoCoffeeQuota, espressoManiacAmericanoCoffeeQuota, espressoManiacCappuccinoCoffeeQuota}
	espressoManiacCoffeeConfig := CoffeeQuotaPerMembership{Membership: coffeedb.EspressoManiac, Quota: espressoManiacQuotas}
	coffeeConfig[coffeedb.EspressoManiac] = espressoManiacCoffeeConfig
	espressoManiacCoffeeConfig.PrintConfig()
}

// InitDb - database initializing
// dbFolder is the folder where user's data will be stored
func InitDb(dbFolder string) {
	var err error
	db, err = coffeedb.Init(dbFolder)
	if err != nil {
		log.Fatal(err)
	}
}

// StartHttpServer starts http server and init api endpoints
func StartHttpServer(server *http.Server, httpHandler *http.ServeMux) {
	log.Println("Starting server")

	httpHandler.Handle("/registerUser", http.HandlerFunc(apiRegisterUser))
	httpHandler.Handle("/buyCoffee", http.HandlerFunc(apiBuyCoffee))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	} else {
		log.Println("server stopped gracefully")
	}
}

// ShutdownHttpServer stops http server
func ShutdownHttpServer(ctx context.Context, server *http.Server) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	} else {
		log.Println("server shutdown")
	}
}
