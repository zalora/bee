package main

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/astaxie/beego/swagger"
	"github.com/rbretecher/go-postman-collection"
)

var (
	description = `# DORAEMON POSTMAN COLLECTION\n## Usage\nPut ` + "`{{DOR_BASE_URL}}`" + `as environment. For more context, refer to: https://learning.postman.com/docs/sending-requests/variables/.`

	baseHeaders = []*postman.Header{
		{
			Key:   "Accept",
			Value: "application/json",
		},
		{
			Key:   "Content-Language",
			Value: "{{DOR_CONTENT_LANGUAGE}}",
		},
		{
			Key:   "User-Agent",
			Value: "{{DOR_USER_AGENT}}",
		},
	}
)

func generatePostman(curpath string) error {
	fd, err := os.ReadFile(path.Join(curpath, "swagger", "swagger.json"))
	if err != nil {
		return err
	}

	var sAPIs swagger.Swagger
	err = json.Unmarshal(fd, &sAPIs)
	if err != nil {
		return err
	}

	p := postman.CreateCollection(sAPIs.Infos.Title, description)
	collection := make(map[string]*postman.Items)

	for sURL, sItem := range sAPIs.Paths {
		sURL = sAPIs.BasePath + sURL

		// Postman uses `:pathVariable` instead of swagger's `{pathVariable}`
		// format.
		sURL = strings.ReplaceAll(sURL, "{", ":")
		sURL = strings.ReplaceAll(sURL, "}", "")

		if get := sItem.Get; get != nil {
			c := upsertNewCollection(p, collection, get.Tags[0])
			addItemToCollection(sURL, c, get, postman.Get)
		}

		if put := sItem.Put; put != nil {
			c := upsertNewCollection(p, collection, put.Tags[0])
			addItemToCollection(sURL, c, put, postman.Put)
		}

		if post := sItem.Post; post != nil {
			c := upsertNewCollection(p, collection, post.Tags[0])
			addItemToCollection(sURL, c, post, postman.Post)
		}

		if del := sItem.Delete; del != nil {
			c := upsertNewCollection(p, collection, del.Tags[0])
			addItemToCollection(sURL, c, del, postman.Delete)
		}

		if options := sItem.Options; options != nil {
			c := upsertNewCollection(p, collection, options.Tags[0])
			addItemToCollection(sURL, c, options, postman.Options)
		}

		if head := sItem.Head; head != nil {
			c := upsertNewCollection(p, collection, head.Tags[0])
			addItemToCollection(sURL, c, head, postman.Head)
		}

		if patch := sItem.Patch; patch != nil {
			c := upsertNewCollection(p, collection, patch.Tags[0])
			addItemToCollection(sURL, c, patch, postman.Patch)
		}
	}

	sort.Slice(p.Items, func(i, j int) bool {
		if strings.EqualFold(p.Items[i].Name, "customers") {
			return true
		}

		return p.Items[i].Name < p.Items[j].Name
	})

	if _, err = json.Marshal(p); err != nil {
		return err
	}

	pd, err := os.Create(path.Join(curpath, "swagger", "postman-collection.json"))
	if err != nil {
		return err
	}
	defer pd.Close()
	err = p.Write(pd, postman.V210)
	if err != nil {
		return err
	}

	return nil
}

func upsertNewCollection(p *postman.Collection, collection map[string]*postman.Items, s string) *postman.Items {
	if c, ok := collection[s]; ok {
		return c
	}

	collection[s] = p.AddItemGroup(s)
	return collection[s]
}

func addItemToCollection(url string, collection *postman.Items, op *swagger.Operation, method postman.Method) {
	var headers []*postman.Header
	var variables []*postman.Variable
	var queryParams []*postman.QueryParam
	var body *postman.Body
	var formData []*postman.Variable
	for _, param := range op.Parameters {
		switch param.In {
		case "path":
			variables = append(variables, &postman.Variable{
				ID:          param.Name,
				Type:        param.Type,
				Name:        param.Name,
				Description: param.Description,
			})
		case "formData":
			formData = append(formData, &postman.Variable{
				Key:         param.Name,
				Type:        param.Type,
				Description: param.Description,
			})
		case "query":
			description := param.Description
			queryParams = append(queryParams, &postman.QueryParam{
				Key:         param.Name,
				Description: &description,
			})
		}
	}

	if len(formData) > 0 {
		body = &postman.Body{
			Mode: "formdata",
		}
		body.FormData = formData
		headers = append(headers, &postman.Header{
			Key:   "Content-Type",
			Value: "multipart/form-data",
		})
	}

	var responses []*postman.Response
	for status, response := range op.Responses {
		s, _ := strconv.Atoi(status)
		responses = append(responses, &postman.Response{
			Status: status,
			Code:   s,
			Name:   response.Description,
		})
	}

	collection.AddItem(postman.CreateItem(postman.Item{
		Name:        string(method) + " " + url,
		Description: op.Description,
		ID:          op.OperationID,
		Request: &postman.Request{
			URL: &postman.URL{
				Host:      []string{"{{DOR_BASE_URL}}"},
				Path:      strings.Split(url, "/"),
				Query:     queryParams,
				Variables: variables,
			},
			Method: method,
			Header: append(baseHeaders, headers...),
			Body:   body,
		},
		Responses: responses,
	}))
}
