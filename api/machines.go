package api

import (
	"net/http"
	"path"

	log "github.com/coreos/fleet/third_party/github.com/golang/glog"

	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/registry"
	"github.com/coreos/fleet/schema"
)

func wireUpMachinesResource(mux *http.ServeMux, prefix string, reg registry.Registry) {
	res := path.Join(prefix, "machines")
	mr := machinesResource{reg}
	mux.Handle(res, &mr)
}

type machinesResource struct {
	reg registry.Registry
}

func (mr *machinesResource) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, err := findNextPageToken(req.URL)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if token == nil {
		def := DefaultPageToken()
		token = &def
	}

	page, err := getMachinePage(mr.reg, *token)
	if err != nil {
		log.Error("Failed fetching page of Machines: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	sendResponse(rw, page)
}

func getMachinePage(reg registry.Registry, tok PageToken) (*schema.MachinePage, error) {
	all, err := reg.GetActiveMachines()
	if err != nil {
		return nil, err
	}

	page := extractPage(all, tok)
	return page, nil
}

func extractPage(all []machine.MachineState, tok PageToken) *schema.MachinePage {
	total := len(all)

	startIndex := int((tok.Page - 1) * tok.Limit)
	stopIndex := int(tok.Page * tok.Limit)

	var items []machine.MachineState
	var next *PageToken

	if startIndex < total {
		if stopIndex > total {
			stopIndex = total
		} else {
			n := tok.Next()
			next = &n
		}

		items = all[startIndex:stopIndex]
	}

	return newMachinePage(items, next)
}

func newMachinePage(items []machine.MachineState, tok *PageToken) *schema.MachinePage {
	smp := schema.MachinePage{
		Machines: make([]*schema.Machine, 0, len(items)),
	}

	if tok != nil {
		smp.NextPageToken = tok.Encode()
	}

	for i, _ := range items {
		ms := items[i]
		sm := mapMachineStateToSchema(&ms)
		smp.Machines = append(smp.Machines, sm)
	}
	return &smp
}

func mapMachineStateToSchema(ms *machine.MachineState) *schema.Machine {
	sm := schema.Machine{
		Id:        ms.ID,
		PrimaryIP: ms.PublicIP,
	}

	sm.Metadata = make(map[string]string, len(ms.Metadata))
	for k, v := range ms.Metadata {
		sm.Metadata[k] = v
	}

	return &sm
}