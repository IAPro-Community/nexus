package web

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestCostCSVPreservesUnknownAndRedactsScope(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR",t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR",t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)
	client,srv,_:=csrfClient(t)
	st:=nexus.Default().Store()
	if st==nil{t.Fatal("store unavailable")}
	cost:=int64(250000)
	_,err:=st.RecordCostObservation(store.CostObservation{
		Provider:"codex", Profile:"work", Model:"model-a", ProjectID:"project-a",
		CostMicroUSD:&cost, CostKind:store.CostKindProviderReported,
		Source:"provider_response", AccountScope:"private-scope",
		ObservedAt:time.Now().UTC(),
	})
	if err!=nil{t.Fatal(err)}
	_,err=st.RecordCostObservation(store.CostObservation{
		Provider:"claude",Profile:"unknown",CostKind:store.CostKindUnknown,
		AccountScope:"another-private-scope",ObservedAt:time.Now().UTC(),
	})
	if err!=nil{t.Fatal(err)}
	res,err:=client.Get(srv.URL()+"/api/v1/cost/observations.csv?days=30")
	if err!=nil{t.Fatal(err)}
	defer res.Body.Close()
	if res.StatusCode!=http.StatusOK{t.Fatalf("unexpected CSV status %d",res.StatusCode)}
	if res.Header.Get("Cache-Control")!="private, no-store"{t.Fatal("CSV must not be cached")}
	records,err:=csv.NewReader(res.Body).ReadAll()
	if err!=nil{t.Fatal(err)}
	if len(records)!=3{t.Fatalf("expected 2 observations + heading, got %v",records)}
	header:=strings.Join(records[0],",")
	if strings.Contains(header,"account_scope"){t.Fatal("account identity must be omitted")}
	if !strings.Contains(header,"cost_microusd"){t.Fatal("cost evidence heading missing")}
	costIndex:=-1
	for i,h:=range records[0]{if h=="cost_microusd"{costIndex=i}}
	if costIndex<0{t.Fatal("cost field missing")}
	known:=false
	unknown:=false
	for _,row:=range records[1:]{
		if row[costIndex]=="250000"{known=true}
		if row[costIndex]==""{unknown=true}
		if strings.Contains(strings.Join(row,","),"private-scope"){t.Fatal("scope leaked")}
	}
	if !known||!unknown{t.Fatalf("unknown cost must remain blank: %+v",records)}
}
