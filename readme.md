# Auction Bid Tracker
You have been asked with building part of a simple online auction system which will allow users to
concurrently bid on items for sale. The system needs to be built in Go and/or Python.

Please, provide a bid-tracker interface and concrete implementation with the following functionality:
- record a user’s bid on an item;
- get the current winning bid for an item;
- get all the bids for an item;
- get all the items on which a user has bid;
- build simple REST API to manage bids.

You are not required to implement a GUI (or CLI) or persistent store (events are for reporting only). You
may use any appropriate libraries to help.

Test the performance of your solution.

Please include the full source code, program parameters & instructions with your solution. Describe
chosen data structures & concurrency approach.

Thank you again for taking the time to complete the task, we are looking forward to your solution!

## Assumptions
I assume the purpose of this task is to test my abilities with concurrency and parallelism, so I have dropped or simplified down all authentication/authorization, and auction administration (item CRUD) logic.

**"build simple REST API to manage bids."**
I assume this item refers to the tasks above it, in the sense that they should be delivered as REST endpoints instead of a console application, and it's not asking to implement additional CRUD logic.

## Simplifications
- Users do not need a password to get an auth token

## Inferred Business Rules:
- Before bidding around, users must visit our super-secure `/login` page to get a JWT token 
- New bids are not recorded for items that already have a higher bid, but they'll persist on user's bid history

## Not included:
- Authentication (anyone can get a token) and Authorization (no granular ACL rules, only basic URL signature)
- Item Create/Update/Delete (they are all predefined in items.json file)

# Tests

## Automated testing

##### Tests performed on a Macbook Pro 2017 - 2,3 GHz Intel Core i5
```bash
# 50 connections
go test -race  2.4s #no race condition detected

# 500 connections
go test        3.7s
go test -v     37.8s
go test -race  #won't run: max goroutines exceeded

# 700 connections
go test        5.8s
go test -v     111.6s

# 1000 connections
go test        9.2s
go test -v     157.6s

# 5000 connections
go test        41.7s
go test -v     #did not try
```

## Manual testing

### Starting the server:
```bash
go run .

# or
go build
./bidtracker

# or
docker image build -t bidtracker .
docker container run -p 3000:3000 bidtracker
```

### Getting a JWT Token and Basic Testing:
**Request:**
```
curl --request POST -d 'username=YOUR_NAME' http://localhost:3000/login
```
**Response:**
```
{"token":"COPY_TOKEN_HERE"}
```
Copy the token for future use. It's valid for 72 hours.

##### Unsigned Requests (public routes)
```
curl http://localhost:3000/items
curl http://localhost:3000/items/1
curl http://localhost:3000/items/1/bids
curl http://localhost:3000/items/1/bids/winning
```

##### Signed Requests (protected routes)
```
curl --request GET http://localhost:3000/items/my_bids -H "Authorization: Bearer PASTE_TOKEN_HERE"
curl --request POST -d 'value=YOUR_BID_VALUE' http://localhost:3000/items/7/bids/create -H "Authorization: Bearer PASTE_TOKEN_HERE"
```

## Stress testing
```
brew install apib
apib -c 100 -d 60 http://localhost:3000/items
apib -f form_input.txt -c 100 -d 60 -H "Authorization: Bearer PASTE_TOKEN_HERE" http://localhost:3000/items/1/bids/create
```
###### Where: 
**-c** is the number of concurrent threads
**-d** is the duration of the requests in seconds

###### More info:
https://github.com/apigee/apib


