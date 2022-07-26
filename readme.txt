Http server starts and listening on port 8080

There are 2 endponts defined: registerUser and buyCoffee

Server when starts it create a folder "Data" where all user's data get stored.
To register a user make a request like:
curl -X POST --data "{\"user_id\":\"user1\", \"membership\":1}" -H "Content-Type: application/json" http://localhost:8080/registerUser

membership could be 1,2,3 which coresponds to Basic, CoffeeLover and EspressoManiac membership type

To use buyCoffee endpont use:
curl -X POST --data "{\"user_id\":\"user1\", \"coffee_type\":1}" -H "Content-Type: application/json" http://localhost:8080/buyCoffee

coffee_type could be 1, 2, 3 which coresponds to Espresso, Americano and Cappuccino coffee type

more testing requests are in curlreq.txt file