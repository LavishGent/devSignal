package main

import (
	"devsignal/internal/httpapi"
	"devsignal/internal/recommend"
	"devsignal/internal/repo/postgres"
	"devsignal/internal/search"
	"devsignal/internal/snapshot"
)

// registerModules attaches every career-record module plus the cross-module
// insight endpoints to the server. New modules are added here and nowhere
// else. Interviews and study sessions take a second repository to verify
// cross-record links belong to the user.
func registerModules(server *httpapi.Server, base httpapi.BaseHandler, store *postgres.Store) {
	server.AddModule(httpapi.NewApplicationsHandler(base, store.Applications))
	server.AddModule(httpapi.NewContactsHandler(base, store.Contacts))
	server.AddModule(httpapi.NewInterviewsHandler(base, store.Interviews, store.Applications))
	server.AddModule(httpapi.NewSkillsHandler(base, store.Skills))
	server.AddModule(httpapi.NewStudySessionsHandler(base, store.StudySessions, store.Skills))
	server.AddModule(httpapi.NewCertificationsHandler(base, store.Certifications))
	server.AddModule(httpapi.NewProjectsHandler(base, store.Projects))
	server.AddModule(httpapi.NewGoalsHandler(base, store.Goals))
	server.AddModule(httpapi.NewLookupsHandler(base))

	loader := snapshot.Loader{
		Applications:   store.Applications,
		Contacts:       store.Contacts,
		Interviews:     store.Interviews,
		Skills:         store.Skills,
		StudySessions:  store.StudySessions,
		Certifications: store.Certifications,
		Projects:       store.Projects,
		Goals:          store.Goals,
	}
	searcher := &search.Service{
		Applications:   store.Applications,
		Contacts:       store.Contacts,
		Interviews:     store.Interviews,
		Skills:         store.Skills,
		StudySessions:  store.StudySessions,
		Certifications: store.Certifications,
		Projects:       store.Projects,
		Goals:          store.Goals,
	}
	server.AddModule(httpapi.NewInsightsHandler(base, loader, recommend.NewEngine(), searcher))
}
