package httpTransport

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/vporoshok/di/example/config"
	"github.com/vporoshok/di/example/model"

	"github.com/go-chi/chi"
	"github.com/vporoshok/di"
)

func MakeServer(ctx context.Context, deps struct {
	DC     di.Container   `di:"#"`
	Config *config.Config `di:"config"`
}) *http.Server {
	router := chi.NewRouter()
	router.Route("/public", func(r chi.Router) {
		r.Post("/users", deps.DC.MustMake(CreateUser).(http.HandlerFunc))
	})
	return &http.Server{
		Addr:        deps.Config.HTTPBind,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     router,
	}
}

func CreateUser(deps struct {
	Action func(context.Context, *model.User) error `di:"create user action"`
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var user *model.User
		if err = json.Unmarshal(body, user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = deps.Action(ctx, user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		je := json.NewEncoder(w)
		_ = je.Encode(user)
	}
}
