package main

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
)

type FileData struct {
	// Id          int
	Transfer    string
	File        string //mets-342432.xml
	Events      []Event
	Agent       string // the preservation system e.g. Archivematica, a3m
	EventCount  int    // e.g. len(events)
	SuccesCount int    // e.g. the amount of event.outcome that are positive or pass
}

type Event struct {
	// Id            string `json:"id"` //uuid type of field taken from the mets
	OutcomeDetail string `json:"outcomeDetail"`
	EventDetail   string `json:"eventDetail"`
	Type          string `json:"type"`    //event type e.g. fixity check, creation
	ObjectName    string `json:"name"`    //premisObjectOrginalName
	Outcome       bool   `json:"outcome"` //can be empty, but this one is weird e.g. pass, Positive, etc.
}

type PremisData struct {
	Events       []string
	SuccessCount int
	EventCount   int
	Agent        string
}

// The complete paths for all the necessary items are known at
// compile time So they can be laid out here. And it should be
// allowable to change in the configuration, but I don't think this is
// a priority feature.
// Get all the amdSecs instead of searching from the root directly search
// through the amdSecPath do this for each section.
var amdSecPath = etree.MustCompilePath("//mets:amdSec")
var eventSecPath = etree.MustCompilePath(".//premis:event")

// I seperate the variables here to give further clarity as to their priority and
// use. The path above is used as the roots for the paths below in my function
// handle function.
var (
	transferNamePath = etree.MustCompilePath("//dcterms:dublincore/dc:identifier")
	objectNamePath   = etree.MustCompilePath(".//premis:object/premis:originalName")
	eventTypePath    = etree.MustCompilePath("./premis:eventType")
	eventAmountPath  = etree.MustCompilePath("//premis:event/premis:eventType")
	// eventId         = etree.MustCompilePath(".//premis:event/premis:eventIdentifierValue")
	agentPath       = etree.MustCompilePath(".//premis:agent/premis:agentIdentifier/premis:agentIdentifierValue")
	eventDetailPath = etree.MustCompilePath("./premis:eventDetailInformation/premis:eventDetail")
	outcomePath     = etree.MustCompilePath("./premis:eventOutcomeInformation/premis:eventOutcome")
	oDetailPath     = etree.MustCompilePath("./premis:eventOutcomeInformation/premis:eventOutcomeDetail/premis:eventOutcomeDetailNote")
)

func (md *FileData) handleEvents(amdSec *etree.Element) {
	var agent string

	// There should only be one objectNameEle
	objectNameEle := amdSec.FindElementPath(objectNamePath)
	// There should only ever be one agent
	agentEles := amdSec.FindElementsPath(agentPath)

	// Get all events from amdSec
	prs := amdSec.FindElementsPath(eventSecPath)

	// Loop through all the elements in the amd section that have been given.
	for _, pr := range prs {
		event := &Event{
			Type:       pr.FindElementPath(eventTypePath).Text(),
			ObjectName: objectNameEle.Text(),
		}

		// Process event details, outcomes, and outcome details
		detailEle := pr.FindElementPath(eventDetailPath)
		outcomeEle := pr.FindElementPath(outcomePath)
		oDetailEle := pr.FindElementPath(oDetailPath)

		if detailEle != nil {
			event.EventDetail = detailEle.Text()
		}

		if outcomeEle != nil {
			outcomeText := strings.ToLower(outcomeEle.Text())
			event.Outcome = strings.Contains(outcomeText, "pass") ||
				strings.Contains(outcomeText, "positive") ||
				strings.Contains(outcomeText, "transcribed") ||
				(outcomeText == "")
		}

		if oDetailEle != nil {
			event.OutcomeDetail = oDetailEle.Text()
		}

		if event.Outcome {
			md.SuccesCount++
		}

		// Append the event to the md.Events slice
		md.Events = append(md.Events, *event)
	}
	// TODO: Create a better abstraction that checks that the identifier
	// type is preservation system and then checks the value
	for _, agentElement := range agentEles {
		if strings.Contains(agentElement.Text(), "Archivematica") ||
			strings.Contains(agentElement.Text(), "a3m") {
			agent = agentElement.Text()
			break
		}
	}
	md.Agent = agent
}

func convertAllEvents(events []Event, agent string) map[string]PremisData {
	dd := make(map[string]PremisData)
	for _, e := range events {
		file := filepath.Base(string(e.ObjectName))
		entry, _ := dd[file]
		entry.Events = append(entry.Events, e.Type)
		entry.EventCount++
		if e.Outcome {
			entry.SuccessCount++
		}
		entry.Agent = agent
		dd[file] = entry
	}
	return dd
}

func serializeEvents(e []Event) ([]byte, error) {
	jsd, err := json.MarshalIndent(e, "", "\t")
	if err != nil {
		return jsd, err
	}

	return jsd, nil
}
