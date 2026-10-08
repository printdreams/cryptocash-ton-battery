package handler

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/printdreams/cryptocash-ton-battery/docs"
	"github.com/printdreams/cryptocash-ton-battery/internal/abuse"
	"github.com/printdreams/cryptocash-ton-battery/internal/audit"
	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/emulate"
	"github.com/printdreams/cryptocash-ton-battery/internal/firebase"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/message"
	"github.com/printdreams/cryptocash-ton-battery/internal/nonce"
	"github.com/printdreams/cryptocash-ton-battery/internal/policy"
	"github.com/printdreams/cryptocash-ton-battery/internal/price"
	"github.com/printdreams/cryptocash-ton-battery/internal/print"
	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
	"github.com/printdreams/cryptocash-ton-battery/internal/ton"
	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
	swagFiles "github.com/swaggo/http-swagger"
)

type Deps struct {
	Firebase      *firebase.Clients
	Nonces        *nonce.Store
	Verifier      tonproof.Config
	JWT           *auth.Issuer
	Users         *user.Store
	Ledger        *ledger.Store
	AdminToken    string
	DevSign       bool
	Relayer       *relayer.Relayer
	Ton           *ton.Client
	RelayerSender *relayer.Sender
	Emulator      *emulate.Client
	MsgCfg        message.Config
	PolCfg        policy.Config
	RelayGasTON   string
	Price         *price.Oracle
	PrintStore    *print.Store
	PrintCfg      print.Config
	Limiter       *abuse.Limiter
	Killswitch    *abuse.Killswitch
	Audit         *audit.Store
}

func SetupRoutes(r *chi.Mux, d Deps) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/swagger/*", swagFiles.Handler())
	r.Get("/health", HealthCheck)

	fbHandler := NewFirebaseHandler(d.Firebase)
	r.Get("/firebase/status", fbHandler.Status)

	catalogHandler := NewCatalogHandler()
	r.Get("/products", catalogHandler.Products)

	priceHandler := NewPriceHandler(d.Price)
	r.Get("/price", priceHandler.Price)

	tpHandler := NewTonProofHandler(d.Nonces, d.Verifier, d.JWT, d.Users)
	r.Get("/ton-proof/payload", tpHandler.Payload)
	r.Post("/ton-proof/check", tpHandler.Check)

	if d.DevSign {
		devHandler := NewDevSignHandler(d.Verifier)
		r.Get("/ton-proof/dev-sign", devHandler.Sign)
		relayerDev := NewRelayerDevHandler(d.RelayerSender)
		r.Post("/relayer/dev-send", relayerDev.Send)
		emulateDev := NewEmulateDevHandler(d.Emulator)
		r.Post("/emulate", emulateDev.Emulate)
		log.Printf("warning: DEV_SIGN enabled — /ton-proof/dev-sign, /relayer/dev-send and /emulate are active (TO BE DELETED)")
	}

	relayerHandler := NewRelayerHandler(d.Relayer, d.Ton)
	r.Get("/relayer/status", relayerHandler.Status)

	accountHandler := NewAccountHandler(d.Users, d.Ledger)
	r.Group(func(pr chi.Router) {
		pr.Use(auth.Authenticator(d.JWT))
		pr.Get("/auth/me", accountHandler.Me)
		pr.Get("/balance", accountHandler.Balance)
		pr.Get("/transactions", accountHandler.Transactions)

		walletEmu := NewWalletEmulateHandler(d.Emulator, d.Ledger, d.MsgCfg, d.PolCfg, d.Price)
		pr.Post("/wallet/emulate", walletEmu.Emulate)

		messageH := NewMessageHandler(d.Emulator, d.Ledger, d.Users, d.RelayerSender, d.MsgCfg, d.PolCfg, d.RelayGasTON, d.Price, d.Limiter, d.Killswitch)
		pr.Post("/message", messageH.Send)

		printH := NewPrintHandler(d.PrintStore, d.Ledger, d.RelayerSender, d.Price, d.PolCfg, d.PrintCfg, d.Limiter, d.Killswitch)
		pr.Post("/print/quote", printH.Quote)
		pr.Post("/print/execute", printH.Execute)
		pr.Get("/print/{id}", printH.Status)
	})

	if d.AdminToken != "" {
		adminHandler := NewAdminHandler(d.Ledger, d.Killswitch, d.Audit)
		r.Group(func(ar chi.Router) {
			ar.Use(AdminAuth(d.AdminToken))
			ar.Post("/admin/users/{id}/credit", adminHandler.Credit)
			ar.Get("/admin/users/{id}", adminHandler.Balance)
			ar.Post("/admin/killswitch", adminHandler.SetKillswitch)
			ar.Get("/admin/killswitch", adminHandler.GetKillswitch)
			ar.Get("/admin/users/{id}/reconcile", adminHandler.Reconcile)
		})
	} else {
		log.Printf("warning: ADMIN_TOKEN not set — admin routes disabled")
	}
}
