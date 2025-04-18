package dashboard

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Routes(router *mux.Router) {
	fs := http.FileServer(http.Dir("./cmd/api/dashboard/assets"))
	router.PathPrefix("/dashboard/assets/").Handler(
		http.StripPrefix("/dashboard/assets/", fs),
	)
}
