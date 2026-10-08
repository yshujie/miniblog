package notionsync

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yshujie/miniblog/internal/miniblog/source"
)

// The schema-only fixture deliberately has no DB. Any page access is a failure.
type schemaOnlyClient struct {
	calls      []string
	mutate     func(string, *source.NotionDataSource)
	readError  error
	beforeRead func(string)
	forbidden  int
}

func (f *schemaOnlyClient) RetrieveDataSource(ctx context.Context, id string) (source.NotionDataSource, error) {
	f.calls = append(f.calls, id)
	if f.beforeRead != nil {
		f.beforeRead(id)
	}
	schema, _ := (&fixtureNotion{}).RetrieveDataSource(ctx, id)
	if f.mutate != nil {
		f.mutate(id, &schema)
	}
	return schema, f.readError
}
func (f *schemaOnlyClient) QueryDataSource(context.Context, string, bool, string) (source.NotionQueryResult, error) {
	f.forbidden++
	return source.NotionQueryResult{}, errors.New("schema-only command must not query pages")
}
func (f *schemaOnlyClient) RetrievePage(context.Context, string) (source.NotionPage, error) {
	f.forbidden++
	return source.NotionPage{}, errors.New("schema-only command must not retrieve pages")
}

func TestSchemaCheckOnlyReadsFiveSchemasAndSuggestsOpaqueIDs(t *testing.T) {
	client := &schemaOnlyClient{}
	result, err := CheckSchemas(context.Background(), client)
	if err != nil || !result.Complete || len(result.Sources) != 5 || len(client.calls) != 5 || client.forbidden != 0 {
		t.Fatalf("unexpected schema check: %+v %v calls=%v", result, err, client.calls)
	}
	for _, item := range result.Sources {
		if item.Status != "valid" || item.ActualDataSourceID != item.SourceID || item.Config == nil || item.Config.StatePropertyID != "state%3Aid" || len(item.Config.StateOptionIDs) != 4 {
			t.Fatal(item)
		}
	}
}

func TestSchemaCheckReportsRequestedAndActualIdentity(t *testing.T) {
	id, wrongID := AllowedSources()[0], AllowedSources()[1]
	for _, clientRejects := range []bool{false, true} {
		client := &schemaOnlyClient{mutate: func(requested string, schema *source.NotionDataSource) {
			if requested == id {
				schema.ID = wrongID
			}
		}}
		if clientRejects {
			client.readError = errors.New("identity mismatch")
		}
		result, err := CheckSchemas(context.Background(), client)
		if err == nil || result.Complete || len(client.calls) != 5 {
			t.Fatal(result, err)
		}
		first := result.Sources[0]
		if first.SourceID != id || first.ActualDataSourceID != wrongID || first.Config != nil {
			t.Fatal(first)
		}
	}
}

func TestSchemaCheckRejectsInvalidIDsAndTypesButContinues(t *testing.T) {
	cases := map[string]func(*source.NotionDataSource){
		"duplicate decoded property": func(s *source.NotionDataSource) {
			p := s.Properties["知识点"]
			p.ID = "state:id"
			s.Properties["知识点"] = p
		},
		"empty property": func(s *source.NotionDataSource) { p := s.Properties["主题"]; p.ID = ""; s.Properties["主题"] = p },
		"long property": func(s *source.NotionDataSource) {
			p := s.Properties["主题"]
			p.ID = strings.Repeat("x", 129)
			s.Properties["主题"] = p
		},
		"duplicate option": func(s *source.NotionDataSource) {
			p := s.Properties["主题"]
			p.Select.Options = append(p.Select.Options, p.Select.Options[0])
			s.Properties["主题"] = p
		},
		"empty option": func(s *source.NotionDataSource) {
			p := s.Properties["主题"]
			p.Select.Options[0].ID = ""
			s.Properties["主题"] = p
		},
		"wrong type": func(s *source.NotionDataSource) {
			p := s.Properties["博客状态"]
			p.Type = "status"
			s.Properties["博客状态"] = p
		},
		"missing state": func(s *source.NotionDataSource) {
			p := s.Properties["博客状态"]
			p.Select.Options = p.Select.Options[:3]
			s.Properties["博客状态"] = p
		},
		"trash": func(s *source.NotionDataSource) { s.InTrash = true },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			client := &schemaOnlyClient{mutate: func(id string, s *source.NotionDataSource) {
				if id == AllowedSources()[0] {
					change(s)
				}
			}}
			result, err := CheckSchemas(context.Background(), client)
			if err == nil || result.Complete || len(client.calls) != 5 || result.Sources[0].Status != "invalid" || result.Sources[1].Status != "valid" {
				t.Fatal(result, err)
			}
		})
	}
}

func TestSchemaReadErrorsNeverExposeTransportOrEnvironmentDetails(t *testing.T) {
	for _, failure := range []error{errors.New("token-password-dsn-SENTINEL"), &source.NotionAPIError{StatusCode: 403, Code: "token-password-dsn-SENTINEL"}, context.Canceled, context.DeadlineExceeded} {
		result, err := CheckSchemas(context.Background(), &schemaOnlyClient{readError: failure})
		data, marshalErr := json.Marshal(result)
		if err == nil || marshalErr != nil || strings.Contains(string(data), "SENTINEL") || strings.Contains(err.Error(), "SENTINEL") {
			t.Fatalf("unsafe failure: %s %v %v", data, err, marshalErr)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &schemaOnlyClient{}
	result, err := CheckSchemas(ctx, client)
	if err == nil || len(client.calls) != 0 || len(result.Sources) != 5 {
		t.Fatal(result, err, client.calls)
	}
	if result, err = CheckSchemas(context.Background(), nil); err == nil || len(result.Sources) != 5 {
		t.Fatal(result, err)
	}
}
