package main

import (
	"aura/cache"
	"aura/config"
	"aura/database"
	"aura/database/migration"
	autodownload "aura/download/auto"
	"aura/jobs"
	"aura/logging"
	"aura/mediaserver"
	"aura/mediux"
	"aura/utils"
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Whether the database was created on this startup
var databaseIsNew bool

func runBootstrap() (success bool) {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "Bootstrap")
	defer ld.Log()
	config.AppLoadingStep = "Bootstrapping Application"

	logAction := ld.AddAction("Application Startup", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)
	defer logAction.Complete()

	success = false

	// Print App Info
	utils.PrintAppStartUpDetails(APP_VERSION, AUTHOR, LICENSE, APP_PORT, APP_NAME)
	config.AppVersion = APP_VERSION

	// Set Umask for file permissions (if needed)
	config.AppLoadingStep = "Setting UMask for File Permissions"
	utils.SetUMask(ctx)

	// Database: Initialize
	// Done before loading the config, as the config is stored in the database
	config.AppLoadingStep = "Initializing Database"
	newDB, dbInitErr := database.Init(ctx)
	if dbInitErr.Message != "" {
		logging.LOGGER.Error().Timestamp().Str("path", config.DatabasePath()).
			Str("help", "Make sure no other program (e.g. a SQLite viewer) has the database open. If you deleted AURA.db, also delete AURA.db-wal and AURA.db-shm next to it.").
			Msgf("Failed to initialize database: %s", dbInitErr.Message)
		return false
	}
	databaseIsNew = newDB
	config.SetStore(database.ConfigStore())
	logging.LOGGER.Info().Timestamp().Bool("new_database", newDB).Str("path", config.DatabasePath()).Msg("Database initialized")

	// Load the config from the database
	config.AppLoadingStep = "Loading Configuration"
	config.Load(ctx)
	logAction.Complete()

	// Warn while AURA_DISABLE_AUTH is set, so it isn't left on by accident
	if config.AuthDisabledByEnv() {
		logging.LOGGER.Warn().Timestamp().Bool("auth_enabled_in_config", config.Current.Auth.Enabled).
			Msgf("Authentication is turned off by %s. Anyone who can reach aura can use it. Remove %s once you have access again.", config.DisableAuthEnv, config.DisableAuthEnv)
	}

	// Print the config details (sanitized)
	config.Current.PrintDetails()

	// If the config is loaded, validate it.
	// A config that isn't set up yet goes straight to setup instead.
	if config.Loaded && !config.Current.IsSetUp() {
		config.Valid = false
		logging.LOGGER.Info().Timestamp().Msg("aura is not set up yet, starting setup")
	} else if config.Loaded {
		config.AppLoadingStep = "Validating Configuration"
		config.Current.Validate(ctx)
	}

	if config.Loaded && config.Valid {
		success = true
	}

	return success
}

func runPreFlight() (success bool) {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "Preflight")
	defer ld.Log()
	config.AppLoadingStep = "Performing Pre-Flight Checks"

	action := ld.AddAction("Checking Services", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, action)
	defer action.Complete()

	success = false
	config.AppFullyLoaded = false

	// Validate Media Server Connection
	config.AppLoadingStep = "Validating Media Server Connection"
	connectionOk, serverName, serverVersion, msErr := mediaserver.TestConnection(ctx, &config.Current.MediaServer)
	if msErr.Message != "" || !connectionOk || serverVersion == "" || serverName == "" {
		config.MediaServerValid = false
		return success
	}
	if config.Current.MediaServer.Type == "Jellyfin" || config.Current.MediaServer.Type == "Emby" || config.Current.MediaServer.Type == "Silo" {
		// Get Admin User for Emby/Jellyfin/Silo
		config.AppLoadingStep = "Retrieving Media Server Admin User"
		ejUserID, initErr := mediaserver.GetAdminUser(ctx, &config.Current.MediaServer)
		if initErr.Message != "" {
			config.MediaServerValid = false
			return success
		} else if ejUserID == "" {
			config.MediaServerValid = false
			logging.LOGGER.Error().Timestamp().Msgf("Failed to retrieve admin user ID from %s server", config.Current.MediaServer.Type)
			return success
		}
		config.Current.MediaServer.UserID = ejUserID
	}
	config.MediaServerName = serverName
	logging.LOGGER.Trace().Timestamp().Str("media_server_name", serverName).
		Str("media_server_version", serverVersion).
		Msg("Media Server connection validated successfully")
	config.MediaServerValid = true

	// Validate MediUX Token
	config.AppLoadingStep = "Validating MediUX Token"
	mediuxTokenValid, mediuxErr := mediux.ValidateToken(ctx, config.Current.Mediux.ApiToken)
	if mediuxErr.Message != "" || !mediuxTokenValid {
		config.MediuxValid = false
		return success
	}

	if config.MediaServerValid || config.MediuxValid {
		success = true
	}

	return success
}

func runWarmup() (success bool) {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "Warmup")

	action := ld.AddAction("Initializing Application", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, action)
	config.AppLoadingStep = "Warming Up Application"

	success = false

	// Cache: Add all MediUX users
	config.AppLoadingStep = "Preloading MediUX Users into Cache"
	mediux.PreloadMediuxUsers(ctx)

	// Cache: Get a list of all items in MediUX that has a set
	config.AppLoadingStep = "Preloading MediUX Items with Sets into Cache"
	mediux.PreLoadMediuxItemsWithSets(ctx)

	// Database-Migration: If not a new DB, run migrations
	// The database is initialized in runBootstrap
	if !databaseIsNew {
		config.AppLoadingStep = "Running Database Migrations"
		migrationsCompleted, _ := migration.RunMigrations()
		logging.LOGGER.Info().Timestamp().Msgf("%d database migrations performed", migrationsCompleted)
	}

	// Cache: Add all media server sections and items
	config.AppLoadingStep = "Preloading Media Server Data into Cache"
	_ = mediaserver.GetAllLibrarySectionsAndItems(ctx, false)
	logging.LOGGER.Info().Timestamp().Int("sections", cache.LibraryStore.GetSectionsCount()).Int("items", cache.LibraryStore.GetItemsCount()).Msg("Loaded Media Server sections and items into cache")
	logging.LOGGER.Info().Timestamp().Int("collection_items", len(cache.CollectionsStore.GetAllCollections())).
		Msg("Loaded Media Server collections into cache")

	// Database: Vacuum
	config.AppLoadingStep = "Optimizing Database"
	vacuumErr := database.Vacuum(ctx)
	if vacuumErr.Message != "" {
		logging.LOGGER.Error().Timestamp().Msgf("Database VACUUM failed: %s", vacuumErr.Message)
		return false
	}

	action.Complete()
	ld.Log()

	// Cronjob: Auto Download Processing
	config.AppLoadingStep = "Starting Background Jobs"
	jobs.StartAutoDownloadJob()

	// Download Queue: Start the worker that processes download queue jobs from the database
	jobs.StartDownloadQueueWorker(context.Background())

	// Cronjob: Refresh Media Items and Collections
	err := jobs.StartRefreshMediaItemsAndCollectionsJob()
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to schedule Refresh Media Items and Collections cron job")
	}

	// Cronjob: Refresh Mediux Users
	err = jobs.StartRefreshMediuxUsersJob()
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to schedule Refresh Mediux Users cron job")
	}

	// Cronjob: Check MediUX Site Link Availability
	err = jobs.StartCheckMediuxSiteLinkJob()
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to schedule Check MediUX Site Link Availability cron job")
	}

	// Cronjob: Start Check for Media Item Changes Job
	err = jobs.StartCheckForMediaItemChangesJob()
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to schedule Check for Media Item Changes cron job")
	}

	// Cronjob: Start Handle Temp Ignored Items Job
	err = jobs.StartHandleTempIgnoredItemsJob()
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to schedule Handle Temp Ignored Items cron job")
	}

	// Cron: Start Jobs Scheduler
	jobs.StartJobs()

	// Check MediUX Site Link Availability immediately on startup
	config.AppLoadingStep = "Checking MediUX Site Link Availability"
	mediux.CheckSiteLinkAvailability()

	// Initialize MediUX WebSocket Listener
	//go autodownload.StartMediuxWebSocketClient()

	// Initialize Media Server WebSocket Listener (if supported)
	autodownload.StartOrRestartPlexWebSocketClient()

	success = true
	return success
}

// The HTTP server, kept so it can be shut down
var apiServer = &http.Server{Handler: http.HandlerFunc(dispatch)}

func startAPI() {
	// Start HTTP Server
	logging.LOGGER.Info().Timestamp().Int("port", APP_PORT).
		Bool("full_routes", config.Loaded && config.Valid).
		Str("log_level", logging.LOGGER.GetLevel().String()).
		Msg("Starting HTTP Server")
	apiServer.Addr = fmt.Sprintf(":%d", APP_PORT)
	if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.LOGGER.Fatal().Err(err).Msg("Failed to start server")
	}
}

// dispatch forwards to the currently active router.
func dispatch(w http.ResponseWriter, r *http.Request) {
	v := activeHandler.Load()
	h, ok := v.(http.Handler)
	if !ok || h == nil {
		// router not initialized yet (or stored value is wrong type)
		logging.LOGGER.Error().
			Timestamp().
			Str("path", r.URL.Path).
			Msg("activeHandler not initialized")
		http.Error(w, "Service starting up; router not initialized", http.StatusServiceUnavailable)
		return
	}

	h.ServeHTTP(w, r)
}
