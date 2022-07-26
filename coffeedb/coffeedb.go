package coffeedb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

type MembershipType uint8
type CoffeeType uint8

const (
	Basic MembershipType = iota + 1
	CoffeeLover
	EspressoManiac
)

const (
	Espresso CoffeeType = iota + 1
	Americano
	Cappuccino
)

func (m MembershipType) String() string {
	return [...]string{"Basic", "Coffee Lover", "Espresso Maniac"}[m-1]
}

func (m MembershipType) EnumIndex() uint8 {
	return uint8(m)
}

func (c CoffeeType) String() string {
	return [...]string{"Espresso", "Americano", "Cappuccino"}[c-1]
}

func (c CoffeeType) EnumIndex() uint8 {
	return uint8(c)
}

func (ct CoffeeType) IsValid() error {
	switch ct {
	case Espresso, Americano, Cappuccino:
		return nil
	}
	return errors.New("invalid coffee type")
}

func (mt MembershipType) IsValid() error {
	switch mt {
	case Basic, CoffeeLover, EspressoManiac:
		return nil
	}
	return errors.New("invalid membership type")
}

// UserCoffeeQuota user data
type UserCoffeeQuota struct {
	AmountBought    uint32 `json:"amount_bought"`
	StartBoughtTime int64  `json:"bought_time"`
}

type UserCoffeeMembership struct {
	Membership MembershipType                 `json:"membership"`
	QuotaState map[CoffeeType]UserCoffeeQuota `json:"quota_state"`
}

type CoffeeDb struct {
	users     map[string]UserCoffeeMembership
	dbDataDir string
	lock      sync.Mutex
}

func (um *UserCoffeeMembership) Print() {
	fmt.Printf("membership %s\n", um.Membership.String())
	for key, value := range um.QuotaState {
		fmt.Printf("coffee type %s\n", key.String())
		fmt.Printf("amount bought %d\n", value.AmountBought)
		fmt.Printf("bought time %d\n", value.StartBoughtTime)
	}
}

// Init initialize db folder where all user's file will be stored
func Init(dbFolder string) (*CoffeeDb, error) {
	pathExecutable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dataDir := path.Dir(pathExecutable) + string(os.PathSeparator) + dbFolder
	if _, err := os.Stat(dataDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dataDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	return &CoffeeDb{users: make(map[string]UserCoffeeMembership), dbDataDir: dataDir}, nil
}

// RegisterUser inserts a new user into db.users and persist user's information on storage
func (db *CoffeeDb) RegisterUser(userId string, membership MembershipType) error {
	if err := db.loadUserDataIfNotExist(userId); errors.Is(err, os.ErrNotExist) {
		db.lock.Lock()
		userData := UserCoffeeMembership{Membership: membership, QuotaState: make(map[CoffeeType]UserCoffeeQuota)}
		db.users[userId] = userData
		db.lock.Unlock()
		return db.saveUserData(userId, &userData)
	}
	return nil
}

func (db *CoffeeDb) loadUserDataIfNotExist(userId string) error {
	data, err := db.readFromStorage(userId)
	if err != nil {
		return err
	}
	var userData UserCoffeeMembership
	err = json.Unmarshal(data, &userData)
	if err != nil {
		fmt.Println(err)
		return err
	}
	db.lock.Lock()
	db.users[userId] = userData
	db.lock.Unlock()
	return nil
}

// userData returns UserCoffeeMembership struct by user id from memory
// if user does not exist - return nil
func (db *CoffeeDb) userData(userId string) *UserCoffeeMembership {
	db.lock.Lock()
	qs, ok := db.users[userId]
	db.lock.Unlock()
	if !ok {
		return nil
	}
	return &qs
}

// GetUserData returns UserCoffeeMembership struct by user id from memory
// if user does not exist in memory then try to load from file
// if file does not exist - return nil
func (db *CoffeeDb) GetUserData(userId string) *UserCoffeeMembership {
	qs := db.userData(userId)
	if qs == nil {
		err := db.loadUserDataIfNotExist(userId)
		if err != nil {
			return nil
		}
		return db.userData(userId)
	}
	return qs
}

func fullUserFileName(userId string) string {
	return userId + ".json"
}

func (db *CoffeeDb) persistOnStorage(userId string, data []byte) error {
	fileName := db.dbDataDir + string(os.PathSeparator) + fullUserFileName(userId)
	err := ioutil.WriteFile(fileName, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (db *CoffeeDb) readFromStorage(userId string) ([]byte, error) {
	fileName := db.dbDataDir + string(os.PathSeparator) + fullUserFileName(userId)
	return ioutil.ReadFile(fileName)
}

func (db *CoffeeDb) saveUserData(userId string, userQuota *UserCoffeeMembership) error {
	data, err := json.Marshal(userQuota)
	if err != nil {
		return err
	}
	return db.persistOnStorage(userId, data)
}

// SetQuotaState sets coffee amount and time of first bought
func (db *CoffeeDb) SetQuotaState(userId string, coffee CoffeeType, qs *UserCoffeeQuota) error {
	if db.GetUserData(userId) != nil {
		db.lock.Lock()
		userQuota := db.users[userId]
		userQuota.QuotaState[coffee] = *qs
		db.lock.Unlock()
		return db.saveUserData(userId, &userQuota)
	}
	return nil
}

// ClearDb remove all files on storage from dbDataDir and
// clears db.users
func (db *CoffeeDb) ClearDb() {
	os.RemoveAll(db.dbDataDir)
	db.lock.Lock()
	defer db.lock.Unlock()
	db.users = make(map[string]UserCoffeeMembership)
}
