# tailcgi

Serve log files with Google OAuth2 authentication and authorization.

You need to create a project at https://console.developers.google.com/apis/credentials to fill the environment
variables `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`.

```
{
  # No need to write to ~/.config/caddy/autosave.json.
  persist_config off

  order authenticate before respond
  order authorize before basicauth
  security {
    oauth identity provider google {
      realm google
      driver google
      client_id {env.GOOGLE_CLIENT_ID}
      client_secret {env.GOOGLE_CLIENT_SECRET}
      scopes openid email profile
    }
    authentication portal portal_login {
      crypto default token lifetime 3600
      enable identity provider google
      # Encode the IP address in the access token.
      enable source ip tracking
      cookie domain example.com
      ui {
        links {
          "My Identity" "/whoami" icon "las la-user"
        }
      }
      # Everyone is a user.
      transform user {
        match realm google
        action add role authp/user
      }
      # Admins.
      transform user {
        match realm google
        match email YOURSELF@gmail.com
        action add role authp/admin
        ui link "Node server logs" https://logs.example.com/webserver.log icon "lab la-node"
        ui link "Caddy logs" https://logs.example.com/example.com.log icon "las la-archive"
      }
    }
    authorization policy policy_adminonly {
      set auth url https://auth.example.com/oauth2/google
      allow roles authp/admin
      validate bearer header
      # Force reauthentication when the IP address changes.
      validate source address
      inject headers with claims
    }
    authorization policy policy_everyone {
      set auth url https://auth.example.com/oauth2/google
      allow roles authp/user
      validate bearer header
      # Force reauthentication when the IP address changes.
      validate source address
      inject headers with claims
    }
  }
}

# Always redirect www. to without www. prefix.
www.example.com {
  log {
    output file logs/www.example.com.log {
      roll_size     100  # Rotate after 100 MB
      roll_keep_for 120d # Keep log files for 120 days
      roll_keep     100  # Keep at most 100 log files
    }
  }
  redir https://example.com{uri} 307
}

# https://auth.example.com/whoami
# https://auth.example.com/portal
auth.example.com {
  log {
    output file logs/auth.example.com.log {
      roll_size     100  # Rotate after 100 MB
      roll_keep_for 120d # Keep log files for 120 days
      roll_keep     100  # Keep at most 100 log files
    }
  }
  authenticate with portal_login
}

# Serve logs
logs.example.com {
  log {
    output file logs/logs.example.com.log {
      roll_size     100  # Rotate after 100 MB
      roll_keep_for 120d # Keep log files for 120 days
      roll_keep     100  # Keep at most 100 log files
    }
  }
  authorize with policy_adminonly
  cgi /* /opt/bin/tailcgi {
    dir logs
    unbuffered_output
  }
}
```

Build caddy with:


```
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
xcaddy build --output caddy \
  --with github.com/aksdb/caddy-cgi/v2 \
  --with github.com/greenpau/caddy-security \
  --with github.com/greenpau/caddy-trace
``
