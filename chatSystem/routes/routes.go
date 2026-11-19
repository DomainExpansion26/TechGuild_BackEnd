package routes

import (
	"techguild-chat/controllers"
	"techguild-chat/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
)

// RegisterCommunicationRoutes registers all 18 ChatSystem endpoints with the Huma API
func RegisterCommunicationRoutes(
	api huma.API,
	redisClient *redis.Client,
	channelCtrl *controllers.ChannelController,
	messageCtrl *controllers.MessageController,
	fileCtrl *controllers.FileController,
	reactionCtrl *controllers.ReactionController,
	pinCtrl *controllers.PinController,
	searchCtrl *controllers.SearchController,
	centrifugoCtrl *controllers.CentrifugoController,
) {
	authMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api, redisClient),
	}

	tags := []string{"Communication"}
	sec := []map[string][]string{{"BearerAuth": {}}}


	// 1. GET /v1/communication/channels — List user's channels (quest/party/dm, grouped by type)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "list-user-channels",
		Method:      "GET",
		Path:        "/v1/communication/channels",
		Tags:        tags,
		Summary:     "List user's channels",
		Description: "Returns all channels (quest, party, dm) that the authenticated user belongs to, grouped by type with member counts and last activity.",
		Middlewares: authMw,
	}, channelCtrl.ListChannelsHandler)

	// 2. POST /v1/communication/channels — Create 1:1 DM channel
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "create-dm-channel",
		Method:      "POST",
		Path:        "/v1/communication/channels",
		Tags:        tags,
		Summary:     "Create 1:1 DM channel",
		Description: "Creates or returns an existing 1:1 direct message channel between the authenticated user and recipient.",
		Middlewares: authMw,
	}, channelCtrl.CreateDMChannelHandler)

	// 3. GET /v1/communication/channels/{id} — Channel info (budget, duration, milestone count, member count)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "get-channel-info",
		Method:      "GET",
		Path:        "/v1/communication/channels/{id}",
		Tags:        tags,
		Summary:     "Get channel info",
		Description: "Returns detailed channel metadata including budget, duration in days, milestone count, and total member count.",
		Middlewares: authMw,
	}, channelCtrl.GetChannelInfoHandler)

	// 4. GET /v1/communication/channels/{id}/members — List channel members with roles
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "list-channel-members",
		Method:      "GET",
		Path:        "/v1/communication/channels/{id}/members",
		Tags:        tags,
		Summary:     "List channel members",
		Description: "Returns all members of the channel along with their assigned roles (leader, member, client).",
		Middlewares: authMw,
	}, channelCtrl.ListChannelMembersHandler)

	// 5. PATCH /v1/communication/channels/{id}/mute — Mute/unmute channel notifications
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "mute-channel",
		Method:      "PATCH",
		Path:        "/v1/communication/channels/{id}/mute",
		Tags:        tags,
		Summary:     "Mute or unmute channel",
		Description: "Updates notification mute preference for the authenticated user in this channel.",
		Middlewares: authMw,
	}, channelCtrl.MuteChannelHandler)

	

	// 6. GET /v1/communication/channels/{id}/messages — Get message history (paginated, cursor-based)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "get-channel-messages",
		Method:      "GET",
		Path:        "/v1/communication/channels/{id}/messages",
		Tags:        tags,
		Summary:     "Get message history",
		Description: "Retrieves paginated, cursor-based message history for the channel with reaction summaries and tombstone handling.",
		Middlewares: authMw,
	}, messageCtrl.GetMessagesHandler)

	// 7. POST /v1/communication/channels/{id}/messages — Send message (REST fallback, supports reply_to_id)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "send-channel-message",
		Method:      "POST",
		Path:        "/v1/communication/channels/{id}/messages",
		Tags:        tags,
		Summary:     "Send message",
		Description: "Persists a new chat message to PostgreSQL and publishes in real time to Centrifugo for connected clients.",
		Middlewares: authMw,
	}, messageCtrl.SendMessageHandler)

	// 8. PATCH /v1/communication/channels/{id}/messages/{msg_id} — Edit message (within 5-minute window)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "edit-channel-message",
		Method:      "PATCH",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}",
		Tags:        tags,
		Summary:     "Edit message (within 5-minute window)",
		Description: "Allows the author to edit message content within 5 minutes of sending. Stores original in metadata for audit continuity.",
		Middlewares: authMw,
	}, messageCtrl.EditMessageHandler)

	// 9. DELETE /v1/communication/channels/{id}/messages/{msg_id} — Delete message (soft delete, tombstone)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "delete-channel-message",
		Method:      "DELETE",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}",
		Tags:        tags,
		Summary:     "Delete message (soft delete, tombstone)",
		Description: "Soft deletes a message leaving a tombstone record for audit continuity and broadcasts the deletion.",
		Middlewares: authMw,
	}, messageCtrl.DeleteMessageHandler)



	// 10. POST /v1/communication/channels/{id}/files — Upload file to channel (pdf,xml,docs,.md)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "upload-channel-file",
		Method:      "POST",
		Path:        "/v1/communication/channels/{id}/files",
		Tags:        tags,
		Summary:     "Upload file to channel",
		Description: "Uploads an attachment (.pdf, .xml, .doc, .docx, .md) with type and size validation and virus scanning hook.",
		Middlewares: authMw,
	}, fileCtrl.UploadFileHandler)



	// 11. POST /v1/communication/channels/{id}/messages/{msg_id}/reactions — Add reaction
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "add-message-reaction",
		Method:      "POST",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}/reactions",
		Tags:        tags,
		Summary:     "Add reaction to message",
		Description: "Adds an emoji reaction to a message. Enforces unique constraint per message per user per emoji.",
		Middlewares: authMw,
	}, reactionCtrl.AddReactionHandler)

	// 12. DELETE /v1/communication/channels/{id}/messages/{msg_id}/reactions/{reaction_id} — Remove reaction
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "remove-message-reaction",
		Method:      "DELETE",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}/reactions/{reaction_id}",
		Tags:        tags,
		Summary:     "Remove reaction from message",
		Description: "Removes an existing emoji reaction added by the authenticated user.",
		Middlewares: authMw,
	}, reactionCtrl.RemoveReactionHandler)


	// 13. POST /v1/communication/channels/{id}/messages/{msg_id}/pin — Pin message
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "pin-channel-message",
		Method:      "POST",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}/pin",
		Tags:        tags,
		Summary:     "Pin message in channel",
		Description: "Marks a message as pinned for quick reference by channel members.",
		Middlewares: authMw,
	}, pinCtrl.PinMessageHandler)

	// 14. DELETE /v1/communication/channels/{id}/messages/{msg_id}/pin — Unpin message
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "unpin-channel-message",
		Method:      "DELETE",
		Path:        "/v1/communication/channels/{id}/messages/{msg_id}/pin",
		Tags:        tags,
		Summary:     "Unpin message in channel",
		Description: "Removes the pinned status from a previously pinned message.",
		Middlewares: authMw,
	}, pinCtrl.UnpinMessageHandler)

	// 15. GET /v1/communication/channels/{id}/pinned — List pinned messages
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "list-pinned-messages",
		Method:      "GET",
		Path:        "/v1/communication/channels/{id}/pinned",
		Tags:        tags,
		Summary:     "List pinned messages",
		Description: "Returns all pinned messages in the channel, indexed for fast retrieval.",
		Middlewares: authMw,
	}, pinCtrl.ListPinnedMessagesHandler)



	// 16. GET /v1/communication/search?q= — Search messages (channel-scoped or global)
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "search-messages",
		Method:      "GET",
		Path:        "/v1/communication/search",
		Tags:        tags,
		Summary:     "Search messages",
		Description: "Performs full-text keyword search across messages strictly scoped to channels the requester is a member of.",
		Middlewares: authMw,
	}, searchCtrl.SearchMessagesHandler)


	// 17. GET /v1/communication/token — Issue Centrifugo connection JWT
	huma.Register(api, huma.Operation{
		Security:    sec,
		OperationID: "get-centrifugo-token",
		Method:      "GET",
		Path:        "/v1/communication/token",
		Tags:        tags,
		Summary:     "Issue Centrifugo connection JWT",
		Description: "Mints a short-lived Centrifugo connection token signed with HMAC SHA-256 for the authenticated user.",
		Middlewares: authMw,
	}, centrifugoCtrl.GetCentrifugoTokenHandler)

	// 18. POST /v1/communication/centrifugo/subscribe-proxy — Called by Centrifugo to check channel subscribe permission
	huma.Register(api, huma.Operation{
		OperationID: "centrifugo-subscribe-proxy",
		Method:      "POST",
		Path:        "/v1/communication/centrifugo/subscribe-proxy",
		Tags:        tags,
		Summary:     "Centrifugo subscribe proxy verification",
		Description: "Internal proxy endpoint called by Centrifugo to verify channel membership before allowing client subscription.",
	}, centrifugoCtrl.CentrifugoSubscribeProxyHandler)
}

