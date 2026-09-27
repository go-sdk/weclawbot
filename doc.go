// Package weclawbot 提供微信 ClawBot 的二维码登录、持续消息接收和文本消息发送能力。
//
// SDK 不持久化 Bot Token、二维码、验证码或消息上下文。调用方负责安全保存登录结果，
// 并在回复消息时把 ReceiveMessages 返回的 from_user_id 和 context_token 传给 SendText。
// 媒体上传、下载、加解密和媒体消息发送不在当前支持范围内。
package weclawbot
