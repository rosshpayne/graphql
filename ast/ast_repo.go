package ast

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	sdl "github.com/graph-sdl/ast"
)

const (
	TableName string = "GraphQL2"
)

type TypeRow struct {
	PKey  string
	SortK string
	Stmt  string
	Type  string //TODO: is this necessary?  Reason: is saves having to parse stmt in order to determine its "type"
}

// cache returns the AST type for a given TypeName
type StmtName_ string
type stmtCache map[StmtName_]StatementDef

type PkRow struct {
	PKey  string
	SortK string
}

var (
	document   string
	defaultDoc string
	stmtCache_ stmtCache
	db         *dynamodb.DynamoDB
)

func init() {
	stmtCache_ = make(stmtCache)

	dynamodbService := func() *dynamodb.DynamoDB {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-1"),
		})
		if err != nil {
			log.Panic(err)
		}
		return dynamodb.New(sess, aws.NewConfig())
	}

	db = dynamodbService()
}

func SetDocument(doc string) {
	document = doc
}

func SetDefaultDoc(doc string) {
	defaultDoc = doc
}

// Fetch - when type is in cache it is said to be "resolved".
//  unresolved types are therefore not in the stmtCaches
// func Fetch(input NameValue_) (StatementDef, bool) {
// 	return CacheFetch(input)
// }

func CacheClear() {
	fmt.Println("******************************************")
	fmt.Println("************ CLEAR CACHE *****************")
	fmt.Println("******************************************")
	stmtCache_ = map[StmtName_]StatementDef{} // map literal to zero cache
}
func CacheFetchStmt(input StmtName_) (StatementDef, bool) {

	if ast, ok := stmtCache_[input]; !ok {
		fmt.Printf("** QL CacheFetchStmt [%s] NOT FOUND \n", input)
		return nil, false
	} else {
		fmt.Printf("** QL CacheFetchStmt [%s] found \n", input)
		return ast, true
	}
}

func Add2StmtCache(input StmtName_, obj StatementDef) {
	fmt.Printf("** Add2Cache  x%sx [%s]\n", input, obj.String())
	stmtCache_[input] = obj
}

func ListCache() []StatementDef {
	l := make([]StatementDef, len(stmtCache_), len(stmtCache_))
	i := 0
	for _, v := range stmtCache_ {
		l[i] = v
		i++
	}
	return l
}

func DBFetch(name sdl.NameValue_) (string, error) {
	//
	// query on recipe name to get RecipeId and  book name
	//
	fmt.Printf("DB Fetch name: [%s]\n", name.String())
	if len(document) == 0 {
		document = defaultDoc
	}
	fmt.Println("DBFetch document : ", document)
	if len(name) == 0 {
		return "", fmt.Errorf("No DB search value provided")
	}
	errmsg := "Error in marshall of pKey "
	//
	pkey := PkRow{PKey: name.String(), SortK: document}
	av, err := dynamodbattribute.MarshalMap(&pkey)
	if err != nil {
		return "", fmt.Errorf("%s. MarshalMap: %s", errmsg, err.Error())
	}
	input := &dynamodb.GetItemInput{
		Key:       av,
		TableName: aws.String(TableName),
	}
	input = input.SetReturnConsumedCapacity("TOTAL").SetConsistentRead(false)
	//
	errmsg = "Error in GetItem "
	result, err := db.GetItem(input)
	if err != nil {
		fmt.Println("ERRORX")
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			//case dynamodb.ErrCodeRequestLimitExceeded:
			//	fmt.Println(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", fmt.Errorf("XX%s %s", errmsg, err.Error())
	}
	fmt.Println("dbFetch: GetItem: Query ConsumedCapacity: \n", result.ConsumedCapacity)

	if len(result.Item) == 0 {
		return "", fmt.Errorf(` No database record found for "%s"`, name)
	}
	rec := &TypeRow{}
	err = dynamodbattribute.UnmarshalMap(result.Item, rec)
	if err != nil {
		fmt.Println(" NO XRECORD FOUND ")
		errmsg := "error in unmarshal "
		return "", fmt.Errorf("%s. UnmarshalMaps:  %s", errmsg, err.Error())
	}
	fmt.Printf("DBfetch result: [%s] \n", rec.Stmt)
	return rec.Stmt, nil
}
