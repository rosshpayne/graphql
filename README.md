# A Graphql Query Server 
GraphQL server - supports Query operation only.  Queries with multiple statements has each statement executed concurrently.

# Testing
cd parser
go test  -v \> test.all.log &
tail -10f test.all.log
