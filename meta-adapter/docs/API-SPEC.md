# API Specification

**Version:** 1.0.0
**Format:** OpenAPI 3.0.3
**Last Updated:** 2025-11-18

## Table of Contents

1. [Overview](#overview)
2. [OpenAPI Specification](#openapi-specification)
3. [Authentication](#authentication)
4. [Rate Limiting](#rate-limiting)
5. [Webhooks](#webhooks)
6. [Error Handling](#error-handling)
7. [API Endpoints Reference](#api-endpoints-reference)
8. [Code Examples](#code-examples)

---

## 1. Overview

This document defines the complete REST API specification for the WhatsApp Meta API Adapter. The API is designed to be **100% compatible** with Meta's official WhatsApp Business Cloud API where possible, while providing additional features for self-hosted deployments.

### Base URL

```
Production:  https://api.whatsapp-adapter.example.com/v1
Development: http://localhost:8080/v1
```

### API Characteristics

- **Protocol:** REST over HTTPS
- **Format:** JSON (UTF-8)
- **Authentication:** OAuth2 + Bearer Tokens
- **Rate Limiting:** Token bucket per tenant
- **Versioning:** URI versioning (/v1, /v2)
- **Idempotency:** Supported via `Idempotency-Key` header

---

## 2. OpenAPI Specification

```yaml
openapi: 3.0.3
info:
  title: WhatsApp Meta API Adapter
  description: |
    Self-hosted WhatsApp Business API compatible with Meta's official API.

    ## Features
    - 100% Meta API compatible message sending
    - Multi-tenant architecture
    - QR code and pairing code authentication
    - Webhook support with retries
    - Media handling (images, videos, documents)
    - OAuth2 authentication

    ## Base URL
    - Production: `https://api.whatsapp-adapter.example.com/v1`
    - Development: `http://localhost:8080/v1`

    ## Rate Limits
    - Free: 100 messages/day, 30 requests/minute
    - Starter: 1,000 messages/day, 60 requests/minute
    - Business: 10,000 messages/day, 120 requests/minute
    - Enterprise: Unlimited, 300 requests/minute

  version: 1.0.0
  contact:
    name: API Support
    email: support@whatsapp-adapter.example.com
    url: https://docs.whatsapp-adapter.example.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: https://api.whatsapp-adapter.example.com/v1
    description: Production server
  - url: http://localhost:8080/v1
    description: Development server

tags:
  - name: Authentication
    description: OAuth2 authentication endpoints
  - name: Instances
    description: WhatsApp instance management
  - name: Messages
    description: Send and receive messages
  - name: Media
    description: Upload and download media files
  - name: Webhooks
    description: Webhook configuration
  - name: Contacts
    description: Contact management
  - name: Groups
    description: Group chat management
  - name: Templates
    description: Message templates
  - name: Health
    description: System health checks

paths:
  # =======================
  # HEALTH CHECK
  # =======================
  /health:
    get:
      tags: [Health]
      summary: Health check
      description: Check if the API is running
      operationId: getHealth
      security: []
      responses:
        '200':
          description: API is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: ok
                  timestamp:
                    type: string
                    format: date-time
                  version:
                    type: string
                    example: 1.0.0

  # =======================
  # AUTHENTICATION
  # =======================
  /oauth/token:
    post:
      tags: [Authentication]
      summary: Get access token
      description: Exchange authorization code for access token (OAuth2)
      operationId: getToken
      security: []
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [grant_type, client_id, client_secret]
              properties:
                grant_type:
                  type: string
                  enum: [authorization_code, refresh_token, client_credentials]
                  example: authorization_code
                code:
                  type: string
                  description: Authorization code (required for authorization_code grant)
                redirect_uri:
                  type: string
                  format: uri
                  description: Redirect URI (required for authorization_code grant)
                refresh_token:
                  type: string
                  description: Refresh token (required for refresh_token grant)
                client_id:
                  type: string
                  example: your_client_id
                client_secret:
                  type: string
                  example: your_client_secret
                scope:
                  type: string
                  example: messages.send messages.read
                code_verifier:
                  type: string
                  description: PKCE code verifier
      responses:
        '200':
          description: Access token granted
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TokenResponse'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'

  /oauth/authorize:
    get:
      tags: [Authentication]
      summary: Authorization endpoint
      description: OAuth2 authorization endpoint (redirects to login)
      operationId: authorize
      security: []
      parameters:
        - name: response_type
          in: query
          required: true
          schema:
            type: string
            enum: [code]
        - name: client_id
          in: query
          required: true
          schema:
            type: string
        - name: redirect_uri
          in: query
          required: true
          schema:
            type: string
            format: uri
        - name: scope
          in: query
          schema:
            type: string
            example: messages.send messages.read
        - name: state
          in: query
          schema:
            type: string
          description: CSRF protection token
        - name: code_challenge
          in: query
          schema:
            type: string
          description: PKCE code challenge
        - name: code_challenge_method
          in: query
          schema:
            type: string
            enum: [S256]
      responses:
        '302':
          description: Redirect to login page or redirect_uri with code
        '400':
          $ref: '#/components/responses/BadRequest'

  # =======================
  # INSTANCES
  # =======================
  /instances:
    get:
      tags: [Instances]
      summary: List instances
      description: Get all WhatsApp instances for the authenticated tenant
      operationId: listInstances
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/Limit'
        - $ref: '#/components/parameters/Offset'
        - name: status
          in: query
          schema:
            type: string
            enum: [connected, disconnected, connecting, failed]
      responses:
        '200':
          description: List of instances
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Instance'
                  paging:
                    $ref: '#/components/schemas/Paging'
        '401':
          $ref: '#/components/responses/Unauthorized'

    post:
      tags: [Instances]
      summary: Create instance
      description: Create a new WhatsApp instance
      operationId: createInstance
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                display_name:
                  type: string
                  example: Customer Support
                webhook_url:
                  type: string
                  format: uri
                  example: https://example.com/webhook
                webhook_events:
                  type: array
                  items:
                    type: string
                  example: [messages, message_status]
      responses:
        '201':
          description: Instance created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Instance'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '402':
          description: Payment required (quota exceeded)
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /instances/{phone_number_id}:
    get:
      tags: [Instances]
      summary: Get instance
      description: Get details of a specific instance
      operationId: getInstance
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '200':
          description: Instance details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Instance'
        '404':
          $ref: '#/components/responses/NotFound'

    patch:
      tags: [Instances]
      summary: Update instance
      description: Update instance configuration
      operationId: updateInstance
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                display_name:
                  type: string
                webhook_url:
                  type: string
                  format: uri
                webhook_events:
                  type: array
                  items:
                    type: string
      responses:
        '200':
          description: Instance updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Instance'
        '400':
          $ref: '#/components/responses/BadRequest'
        '404':
          $ref: '#/components/responses/NotFound'

    delete:
      tags: [Instances]
      summary: Delete instance
      description: Delete an instance (disconnects and removes)
      operationId: deleteInstance
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '204':
          description: Instance deleted
        '404':
          $ref: '#/components/responses/NotFound'

  /instances/{phone_number_id}/qrcode:
    get:
      tags: [Instances]
      summary: Get QR code
      description: Generate QR code for WhatsApp pairing
      operationId: getQRCode
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '200':
          description: QR code generated
          content:
            application/json:
              schema:
                type: object
                properties:
                  qr_code:
                    type: string
                    description: Base64-encoded PNG QR code
                    example: data:image/png;base64,iVBORw0KGgoAAAANS...
                  expires_at:
                    type: string
                    format: date-time
                    example: '2025-11-18T12:05:00Z'
                  pairing_url:
                    type: string
                    example: https://wa.me/qr/ABC123
        '400':
          description: Instance already connected
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /instances/{phone_number_id}/pairing-code:
    post:
      tags: [Instances]
      summary: Get pairing code
      description: Generate 8-digit pairing code for WhatsApp
      operationId: getPairingCode
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [phone_number]
              properties:
                phone_number:
                  type: string
                  description: Phone number to pair (E.164 format)
                  example: '+5511999999999'
      responses:
        '200':
          description: Pairing code generated
          content:
            application/json:
              schema:
                type: object
                properties:
                  pairing_code:
                    type: string
                    example: '12345678'
                  expires_at:
                    type: string
                    format: date-time
        '400':
          $ref: '#/components/responses/BadRequest'

  /instances/{phone_number_id}/disconnect:
    post:
      tags: [Instances]
      summary: Disconnect instance
      description: Logout from WhatsApp (keeps instance)
      operationId: disconnectInstance
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '200':
          description: Instance disconnected
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                    example: true
                  message:
                    type: string
                    example: Instance disconnected successfully

  # =======================
  # MESSAGES
  # =======================
  /{phone_number_id}/messages:
    post:
      tags: [Messages]
      summary: Send message
      description: Send a message (Meta API compatible)
      operationId: sendMessage
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SendMessageRequest'
            examples:
              textMessage:
                summary: Text message
                value:
                  messaging_product: whatsapp
                  recipient_type: individual
                  to: '5511999999999'
                  type: text
                  text:
                    body: Hello, this is a test message!
              imageMessage:
                summary: Image message
                value:
                  messaging_product: whatsapp
                  to: '5511999999999'
                  type: image
                  image:
                    link: https://example.com/image.jpg
                    caption: Check out this image
              templateMessage:
                summary: Template message
                value:
                  messaging_product: whatsapp
                  to: '5511999999999'
                  type: template
                  template:
                    name: hello_world
                    language:
                      code: en_US
      responses:
        '200':
          description: Message sent
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SendMessageResponse'
        '400':
          $ref: '#/components/responses/BadRequest'
        '429':
          $ref: '#/components/responses/RateLimitExceeded'

    get:
      tags: [Messages]
      summary: List messages
      description: Get message history for an instance
      operationId: listMessages
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
        - $ref: '#/components/parameters/Limit'
        - $ref: '#/components/parameters/Offset'
        - name: from
          in: query
          description: Filter by sender phone number
          schema:
            type: string
        - name: to
          in: query
          description: Filter by recipient phone number
          schema:
            type: string
        - name: direction
          in: query
          schema:
            type: string
            enum: [inbound, outbound]
        - name: start_time
          in: query
          schema:
            type: string
            format: date-time
        - name: end_time
          in: query
          schema:
            type: string
            format: date-time
      responses:
        '200':
          description: Message list
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Message'
                  paging:
                    $ref: '#/components/schemas/Paging'

  /{phone_number_id}/messages/{message_id}:
    get:
      tags: [Messages]
      summary: Get message
      description: Get details of a specific message
      operationId: getMessage
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
        - name: message_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Message details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Message'
        '404':
          $ref: '#/components/responses/NotFound'

  # =======================
  # MEDIA
  # =======================
  /{phone_number_id}/media:
    post:
      tags: [Media]
      summary: Upload media
      description: Upload media file (image, video, audio, document)
      operationId: uploadMedia
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              required: [file]
              properties:
                file:
                  type: string
                  format: binary
                  description: Media file to upload
                messaging_product:
                  type: string
                  default: whatsapp
      responses:
        '200':
          description: Media uploaded
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                    example: '1234567890'
        '400':
          $ref: '#/components/responses/BadRequest'
        '413':
          description: File too large
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /{phone_number_id}/media/{media_id}:
    get:
      tags: [Media]
      summary: Get media metadata
      description: Get media file metadata (URL, MIME type, size)
      operationId: getMedia
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
        - name: media_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Media metadata
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Media'
        '404':
          $ref: '#/components/responses/NotFound'

  /media/{media_id}/download:
    get:
      tags: [Media]
      summary: Download media
      description: Download media file (returns binary data)
      operationId: downloadMedia
      security:
        - BearerAuth: []
      parameters:
        - name: media_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Media file
          content:
            image/*:
              schema:
                type: string
                format: binary
            video/*:
              schema:
                type: string
                format: binary
            audio/*:
              schema:
                type: string
                format: binary
            application/pdf:
              schema:
                type: string
                format: binary
        '404':
          $ref: '#/components/responses/NotFound'

  # =======================
  # WEBHOOKS
  # =======================
  /webhooks:
    get:
      tags: [Webhooks]
      summary: Webhook verification
      description: Webhook verification endpoint (Meta compatible)
      operationId: verifyWebhook
      security: []
      parameters:
        - name: hub.mode
          in: query
          required: true
          schema:
            type: string
            example: subscribe
        - name: hub.challenge
          in: query
          required: true
          schema:
            type: string
        - name: hub.verify_token
          in: query
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Verification successful
          content:
            text/plain:
              schema:
                type: string
                description: Returns hub.challenge value
        '403':
          description: Verification failed

  /instances/{phone_number_id}/webhook:
    post:
      tags: [Webhooks]
      summary: Test webhook
      description: Send test webhook to configured URL
      operationId: testWebhook
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '200':
          description: Webhook test sent
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                  status_code:
                    type: integer
                  response_time_ms:
                    type: integer

  # =======================
  # CONTACTS
  # =======================
  /{phone_number_id}/contacts:
    get:
      tags: [Contacts]
      summary: List contacts
      description: Get all contacts for an instance
      operationId: listContacts
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
        - $ref: '#/components/parameters/Limit'
        - $ref: '#/components/parameters/Offset'
        - name: search
          in: query
          schema:
            type: string
          description: Search by name or phone
      responses:
        '200':
          description: Contact list
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Contact'
                  paging:
                    $ref: '#/components/schemas/Paging'

    post:
      tags: [Contacts]
      summary: Create contact
      description: Add or update a contact
      operationId: createContact
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [phone]
              properties:
                phone:
                  type: string
                  example: '+5511999999999'
                first_name:
                  type: string
                last_name:
                  type: string
                email:
                  type: string
                labels:
                  type: array
                  items:
                    type: string
      responses:
        '201':
          description: Contact created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Contact'

  # =======================
  # GROUPS
  # =======================
  /{phone_number_id}/groups:
    get:
      tags: [Groups]
      summary: List groups
      description: Get all groups for an instance
      operationId: listGroups
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
        - $ref: '#/components/parameters/Limit'
        - $ref: '#/components/parameters/Offset'
      responses:
        '200':
          description: Group list
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Group'
                  paging:
                    $ref: '#/components/schemas/Paging'

  # =======================
  # TEMPLATES
  # =======================
  /{phone_number_id}/message_templates:
    get:
      tags: [Templates]
      summary: List templates
      description: Get all message templates
      operationId: listTemplates
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      responses:
        '200':
          description: Template list
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Template'

    post:
      tags: [Templates]
      summary: Create template
      description: Create a new message template
      operationId: createTemplate
      security:
        - BearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PhoneNumberId'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TemplateCreate'
      responses:
        '201':
          description: Template created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Template'

# =======================
# COMPONENTS
# =======================
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: OAuth2 access token

    OAuth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: /v1/oauth/authorize
          tokenUrl: /v1/oauth/token
          scopes:
            messages.send: Send messages
            messages.read: Read message history
            instances.manage: Manage instances
            contacts.read: Read contacts
            contacts.write: Modify contacts
            groups.read: Read group information
            media.upload: Upload media files
            media.download: Download media files
            templates.manage: Manage templates
            webhooks.configure: Configure webhooks

  parameters:
    PhoneNumberId:
      name: phone_number_id
      in: path
      required: true
      description: WhatsApp instance phone number ID
      schema:
        type: string
      example: '1234567890'

    Limit:
      name: limit
      in: query
      description: Maximum number of results
      schema:
        type: integer
        minimum: 1
        maximum: 100
        default: 20

    Offset:
      name: offset
      in: query
      description: Number of results to skip
      schema:
        type: integer
        minimum: 0
        default: 0

  schemas:
    # Authentication
    TokenResponse:
      type: object
      properties:
        access_token:
          type: string
          example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
        token_type:
          type: string
          example: Bearer
        expires_in:
          type: integer
          example: 3600
        refresh_token:
          type: string
          example: def50200a1b2c3d4e5f6...
        scope:
          type: string
          example: messages.send messages.read

    # Instance
    Instance:
      type: object
      properties:
        id:
          type: string
          format: uuid
        phone_number_id:
          type: string
          example: '1234567890'
        phone_number:
          type: string
          example: '+5511999999999'
        display_name:
          type: string
          example: Customer Support
        status:
          type: string
          enum: [connected, disconnected, connecting, failed]
        connection_state:
          type: string
          enum: [connected, disconnected, connecting, logged_out]
        webhook_url:
          type: string
          format: uri
        webhook_events:
          type: array
          items:
            type: string
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
        last_seen_at:
          type: string
          format: date-time

    # Messages
    SendMessageRequest:
      type: object
      required: [messaging_product, to, type]
      properties:
        messaging_product:
          type: string
          enum: [whatsapp]
          default: whatsapp
        recipient_type:
          type: string
          enum: [individual, group]
          default: individual
        to:
          type: string
          description: WhatsApp ID or phone number (without +)
          example: '5511999999999'
        type:
          type: string
          enum: [text, image, video, audio, document, sticker, location, contacts, reaction, interactive, template]
        context:
          type: object
          properties:
            message_id:
              type: string
              description: Message ID to reply to
        text:
          type: object
          properties:
            preview_url:
              type: boolean
              default: false
            body:
              type: string
              maxLength: 4096
        image:
          $ref: '#/components/schemas/MediaContent'
        video:
          $ref: '#/components/schemas/MediaContent'
        audio:
          $ref: '#/components/schemas/MediaContent'
        document:
          $ref: '#/components/schemas/MediaContent'
        location:
          type: object
          properties:
            latitude:
              type: number
              format: double
            longitude:
              type: number
              format: double
            name:
              type: string
            address:
              type: string
        contacts:
          type: array
          items:
            $ref: '#/components/schemas/ContactCard'
        reaction:
          type: object
          properties:
            message_id:
              type: string
            emoji:
              type: string
              example: '👍'
        template:
          type: object
          properties:
            name:
              type: string
            language:
              type: object
              properties:
                code:
                  type: string
                  example: en_US
            components:
              type: array
              items:
                type: object

    MediaContent:
      type: object
      properties:
        id:
          type: string
          description: Media ID from upload
        link:
          type: string
          format: uri
          description: Public URL (alternative to id)
        caption:
          type: string
        filename:
          type: string

    ContactCard:
      type: object
      properties:
        name:
          type: object
          properties:
            formatted_name:
              type: string
            first_name:
              type: string
            last_name:
              type: string
        phones:
          type: array
          items:
            type: object
            properties:
              phone:
                type: string
              type:
                type: string
                enum: [CELL, MAIN, WORK, HOME]

    SendMessageResponse:
      type: object
      properties:
        messaging_product:
          type: string
          example: whatsapp
        contacts:
          type: array
          items:
            type: object
            properties:
              input:
                type: string
              wa_id:
                type: string
        messages:
          type: array
          items:
            type: object
            properties:
              id:
                type: string
                example: wamid.HBgNNTUxMTk5OTk5OTk5ORUCABIYFjNBQjRGMzY3RTcyRTNFRDY1RTlCAAA=

    Message:
      type: object
      properties:
        id:
          type: string
        message_id:
          type: string
        wamid:
          type: string
        from:
          type: string
        to:
          type: string
        direction:
          type: string
          enum: [inbound, outbound]
        type:
          type: string
        status:
          type: string
          enum: [pending, sent, delivered, read, failed]
        content:
          type: object
        timestamp:
          type: string
          format: date-time
        sent_at:
          type: string
          format: date-time
        delivered_at:
          type: string
          format: date-time
        read_at:
          type: string
          format: date-time

    # Media
    Media:
      type: object
      properties:
        id:
          type: string
        url:
          type: string
          format: uri
          description: Download URL (with bearer token)
        mime_type:
          type: string
          example: image/jpeg
        sha256:
          type: string
        file_size:
          type: integer
        width:
          type: integer
        height:
          type: integer

    # Contact
    Contact:
      type: object
      properties:
        id:
          type: string
          format: uuid
        jid:
          type: string
        phone:
          type: string
        display_name:
          type: string
        first_name:
          type: string
        last_name:
          type: string
        email:
          type: string
        labels:
          type: array
          items:
            type: string
        blocked:
          type: boolean

    # Group
    Group:
      type: object
      properties:
        id:
          type: string
          format: uuid
        jid:
          type: string
        name:
          type: string
        description:
          type: string
        participant_count:
          type: integer
        created_at:
          type: string
          format: date-time

    # Template
    Template:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        language:
          type: string
        category:
          type: string
          enum: [marketing, utility, authentication]
        status:
          type: string
          enum: [draft, pending, approved, rejected]
        components:
          type: array
          items:
            type: object

    TemplateCreate:
      type: object
      required: [name, language, category, components]
      properties:
        name:
          type: string
        language:
          type: string
        category:
          type: string
          enum: [marketing, utility, authentication]
        components:
          type: array
          items:
            type: object
            properties:
              type:
                type: string
                enum: [HEADER, BODY, FOOTER, BUTTONS]
              format:
                type: string
                enum: [TEXT, IMAGE, VIDEO, DOCUMENT]
              text:
                type: string

    # Pagination
    Paging:
      type: object
      properties:
        cursors:
          type: object
          properties:
            before:
              type: string
            after:
              type: string
        next:
          type: string
          format: uri
        previous:
          type: string
          format: uri

    # Error
    Error:
      type: object
      properties:
        error:
          type: object
          properties:
            message:
              type: string
              example: Invalid request
            type:
              type: string
              example: ValidationError
            code:
              type: integer
              example: 400
            error_data:
              type: object
              properties:
                details:
                  type: string
            fbtrace_id:
              type: string
              description: Trace ID for debugging

  responses:
    BadRequest:
      description: Bad request
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'

    Unauthorized:
      description: Unauthorized
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error:
              message: Invalid access token
              type: OAuthException
              code: 190

    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error:
              message: Resource not found
              type: NotFoundError
              code: 404

    RateLimitExceeded:
      description: Rate limit exceeded
      headers:
        X-RateLimit-Limit:
          schema:
            type: integer
          description: Request limit per window
        X-RateLimit-Remaining:
          schema:
            type: integer
          description: Remaining requests in window
        X-RateLimit-Reset:
          schema:
            type: integer
          description: Unix timestamp when limit resets
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'
          example:
            error:
              message: Rate limit exceeded
              type: RateLimitError
              code: 429

security:
  - BearerAuth: []
```

---

## 3. Authentication

### 3.1 OAuth2 Flow

```mermaid
sequenceDiagram
    participant User
    participant App
    participant AuthServer
    participant API

    User->>App: Click "Connect WhatsApp"
    App->>AuthServer: GET /oauth/authorize?client_id=...
    AuthServer->>User: Redirect to login
    User->>AuthServer: Enter credentials
    AuthServer->>App: Redirect with code
    App->>AuthServer: POST /oauth/token (code)
    AuthServer->>App: Return access_token
    App->>API: GET /instances (Bearer token)
    API->>App: Return data
```

### 3.2 Authorization Code Grant (PKCE)

```http
GET /v1/oauth/authorize?
  response_type=code&
  client_id=YOUR_CLIENT_ID&
  redirect_uri=https://yourapp.com/callback&
  scope=messages.send+messages.read&
  state=RANDOM_STATE&
  code_challenge=BASE64URL(SHA256(code_verifier))&
  code_challenge_method=S256
```

Exchange code for token:

```http
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=AUTH_CODE&
redirect_uri=https://yourapp.com/callback&
client_id=YOUR_CLIENT_ID&
client_secret=YOUR_CLIENT_SECRET&
code_verifier=ORIGINAL_VERIFIER
```

### 3.3 Access Token Usage

Include in `Authorization` header:

```http
GET /v1/instances
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## 4. Rate Limiting

### 4.1 Rate Limit Headers

Every API response includes:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1700000000
```

### 4.2 Rate Limits by Tier

| Tier | API Requests/min | Messages/day | Webhooks/min |
|------|-----------------|--------------|--------------|
| Free | 30 | 100 | 10 |
| Starter | 60 | 1,000 | 30 |
| Business | 120 | 10,000 | 60 |
| Enterprise | 300 | Unlimited | 120 |

### 4.3 Rate Limit Exceeded Response

```json
{
  "error": {
    "message": "Rate limit exceeded. Try again in 23 seconds.",
    "type": "RateLimitError",
    "code": 429,
    "error_data": {
      "reset_at": "2025-11-18T12:05:00Z"
    }
  }
}
```

---

## 5. Webhooks

### 5.1 Webhook Events

Subscribe to events via `webhook_events` when creating/updating instance:

```json
{
  "webhook_url": "https://yourapp.com/webhook",
  "webhook_events": [
    "messages",
    "message_status",
    "message_read",
    "presence",
    "contacts",
    "groups"
  ]
}
```

### 5.2 Webhook Payload Format (Meta Compatible)

```json
{
  "object": "whatsapp_business_account",
  "entry": [
    {
      "id": "INSTANCE_ID",
      "changes": [
        {
          "value": {
            "messaging_product": "whatsapp",
            "metadata": {
              "display_phone_number": "5511999999999",
              "phone_number_id": "1234567890"
            },
            "contacts": [
              {
                "profile": {
                  "name": "John Doe"
                },
                "wa_id": "5511888888888"
              }
            ],
            "messages": [
              {
                "from": "5511888888888",
                "id": "wamid.HBgNNTUxMTg4ODg4ODg4OBUCABIYFjNBQjRGMzY3RTcyRTNFRDY1RTlCAAA=",
                "timestamp": "1700000000",
                "type": "text",
                "text": {
                  "body": "Hello!"
                }
              }
            ]
          },
          "field": "messages"
        }
      ]
    }
  ]
}
```

### 5.3 Webhook Signature Verification

All webhooks include `X-Hub-Signature-256` header:

```http
X-Hub-Signature-256: sha256=a1b2c3d4e5f6...
```

Verify using your `webhook_secret`:

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const hash = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  return `sha256=${hash}` === signature;
}
```

### 5.4 Webhook Retry Strategy

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1 minute |
| 3 | 5 minutes |
| 4 | 30 minutes |
| 5 | 2 hours |
| 6 | 12 hours |
| 7 | 24 hours (then abandon) |

---

## 6. Error Handling

### 6.1 Error Response Format

All errors follow Meta's format:

```json
{
  "error": {
    "message": "Human-readable error description",
    "type": "ErrorType",
    "code": 400,
    "error_data": {
      "details": "Additional context"
    },
    "fbtrace_id": "trace-12345"
  }
}
```

### 6.2 Common Error Codes

| Code | Type | Description |
|------|------|-------------|
| 400 | ValidationError | Invalid request parameters |
| 401 | AuthenticationError | Invalid or expired token |
| 403 | PermissionError | Insufficient permissions |
| 404 | NotFoundError | Resource not found |
| 409 | ConflictError | Resource already exists |
| 429 | RateLimitError | Rate limit exceeded |
| 500 | InternalError | Server error |
| 503 | ServiceUnavailable | Service temporarily unavailable |

### 6.3 WhatsApp-Specific Errors

| Code | Description |
|------|-------------|
| 131026 | Message undeliverable (recipient blocked sender) |
| 131047 | Re-engagement message required |
| 131051 | Unsupported message type |
| 133000 | Phone number not registered on WhatsApp |
| 133004 | Message template not found |
| 133005 | Template parameter count mismatch |

---

## 7. API Endpoints Reference

### Quick Reference Table

| Method | Endpoint | Description |
|--------|----------|-------------|
| **Authentication** |
| POST | `/oauth/token` | Get access token |
| GET | `/oauth/authorize` | Authorization endpoint |
| **Instances** |
| GET | `/instances` | List instances |
| POST | `/instances` | Create instance |
| GET | `/instances/{id}` | Get instance |
| PATCH | `/instances/{id}` | Update instance |
| DELETE | `/instances/{id}` | Delete instance |
| GET | `/instances/{id}/qrcode` | Get QR code |
| POST | `/instances/{id}/pairing-code` | Get pairing code |
| POST | `/instances/{id}/disconnect` | Disconnect |
| **Messages** |
| POST | `/{phone_id}/messages` | Send message |
| GET | `/{phone_id}/messages` | List messages |
| GET | `/{phone_id}/messages/{msg_id}` | Get message |
| **Media** |
| POST | `/{phone_id}/media` | Upload media |
| GET | `/{phone_id}/media/{media_id}` | Get media metadata |
| GET | `/media/{media_id}/download` | Download media |
| **Contacts** |
| GET | `/{phone_id}/contacts` | List contacts |
| POST | `/{phone_id}/contacts` | Create contact |
| **Groups** |
| GET | `/{phone_id}/groups` | List groups |
| **Templates** |
| GET | `/{phone_id}/message_templates` | List templates |
| POST | `/{phone_id}/message_templates` | Create template |

---

## 8. Code Examples

### 8.1 Send Text Message (Node.js)

```javascript
const axios = require('axios');

async function sendMessage() {
  const response = await axios.post(
    'https://api.example.com/v1/1234567890/messages',
    {
      messaging_product: 'whatsapp',
      to: '5511999999999',
      type: 'text',
      text: {
        body: 'Hello from WhatsApp API!'
      }
    },
    {
      headers: {
        'Authorization': 'Bearer YOUR_ACCESS_TOKEN',
        'Content-Type': 'application/json'
      }
    }
  );

  console.log('Message sent:', response.data);
}
```

### 8.2 Send Image with Caption (Python)

```python
import requests

def send_image():
    url = 'https://api.example.com/v1/1234567890/messages'
    headers = {
        'Authorization': 'Bearer YOUR_ACCESS_TOKEN',
        'Content-Type': 'application/json'
    }
    data = {
        'messaging_product': 'whatsapp',
        'to': '5511999999999',
        'type': 'image',
        'image': {
            'link': 'https://example.com/image.jpg',
            'caption': 'Check out this image!'
        }
    }

    response = requests.post(url, json=data, headers=headers)
    print(response.json())
```

### 8.3 Upload and Send Media (cURL)

```bash
# Step 1: Upload media
curl -X POST https://api.example.com/v1/1234567890/media \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/image.jpg" \
  -F "messaging_product=whatsapp"

# Response: {"id": "MEDIA_ID"}

# Step 2: Send message with media ID
curl -X POST https://api.example.com/v1/1234567890/messages \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "to": "5511999999999",
    "type": "image",
    "image": {
      "id": "MEDIA_ID",
      "caption": "Uploaded image"
    }
  }'
```

### 8.4 Handle Webhook (Express.js)

```javascript
const express = require('express');
const crypto = require('crypto');

const app = express();
app.use(express.json());

// Webhook verification
app.get('/webhook', (req, res) => {
  const mode = req.query['hub.mode'];
  const token = req.query['hub.verify_token'];
  const challenge = req.query['hub.challenge'];

  if (mode === 'subscribe' && token === 'YOUR_VERIFY_TOKEN') {
    res.status(200).send(challenge);
  } else {
    res.sendStatus(403);
  }
});

// Receive webhooks
app.post('/webhook', (req, res) => {
  // Verify signature
  const signature = req.headers['x-hub-signature-256'];
  const body = JSON.stringify(req.body);
  const hash = crypto
    .createHmac('sha256', 'YOUR_WEBHOOK_SECRET')
    .update(body)
    .digest('hex');

  if (`sha256=${hash}` !== signature) {
    return res.sendStatus(403);
  }

  // Process webhook
  const { object, entry } = req.body;

  entry.forEach(item => {
    item.changes.forEach(change => {
      if (change.field === 'messages') {
        const { messages } = change.value;
        messages.forEach(message => {
          console.log('Received message:', message);
          // Process message
        });
      }
    });
  });

  res.sendStatus(200);
});

app.listen(3000);
```

### 8.5 Get QR Code for Pairing (Go)

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func getQRCode(phoneNumberID, accessToken string) {
    url := fmt.Sprintf("https://api.example.com/v1/instances/%s/qrcode", phoneNumberID)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+accessToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var result map[string]interface{}
    json.Unmarshal(body, &result)

    qrCode := result["qr_code"].(string)
    fmt.Println("QR Code:", qrCode)
    // Display QR code to user
}
```

---

**End of API Specification Document**
