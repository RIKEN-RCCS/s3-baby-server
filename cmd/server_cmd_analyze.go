// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	"os"
	"s3-baby-server/internal/server"

	"github.com/spf13/cobra"
)

var addr, logPath, authKey string
var rootCmd = &cobra.Command{
	Use:   "s3-baby-server",
	Short: "The server startup command is incorrect",
}
var serverCmd = &cobra.Command{ // サーバー起動コマンドの定義
	Use:   "serve PATH",
	Short: "File server that serves the specified path via HTTP",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		if env := os.Getenv("AUTH_KEY"); env != "" {
			authKey = env
		} else if authKey == "" {
			authKey = "admin,admin"
		}
	},
	Run: func(_ *cobra.Command, args []string) {
		RootDir := args[0]
		server.Start(RootDir, addr, logPath, authKey)
	},
}

func Execute() {
	if err := serverCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() { // オプションとして許容するコマンド
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&addr, "addr", "127.0.0.1:9000", "IPaddress:Port: Port to bind server to (default 127.0.0.1:9000)")
	serverCmd.Flags().StringVar(&logPath, "logPath", "", "Log output path")
	serverCmd.Flags().StringVar(&authKey, "auth-key", "", "Set key pair: access_key_id,secret_access_key") // rcloneの仕様に準拠
}
