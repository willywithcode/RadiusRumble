extends Node

const packets := preload("res://packets.gd")
const Log := preload("res://classes/log/log.gd")
@onready var _log := $Log as Log
@onready var _line_edit: LineEdit = $LineEdit

func _ready() -> void:
	WS.connected_to_server.connect(_on_ws_connected_to_server)
	WS.connection_closed.connect(_on_ws_connection_closed)
	WS.packet_received.connect(_on_ws_packet_received)
	_line_edit.text_submitted.connect(_on_line_edit_text_submitted)

	_log.info("Connecting to server...")
	WS.connect_to_url("ws://127.0.0.1:8080/ws")

func _on_ws_connected_to_server() -> void:
	_log.success("Connected to server")

func _on_ws_connection_closed(was_clean: bool) -> void:
	_log.warning("Connection closed, clean: %s" % was_clean)

func _on_ws_packet_received(packet: packets.Packet) -> void:
	var senderid = packet.get_sender_id()
	if packet.has_id():
		_handle_id_msg(senderid, packet.get_id())
	elif packet.has_chat():
		_handle_chat_msg(senderid, packet.get_chat())

func _handle_id_msg(_senderid: int, packet: packets.IdMessage) -> void:
	var client_id = packet.get_id()
	_log.info("Received id packet from %s" % client_id)

func _handle_chat_msg(senderid: int, packet: packets.ChatMessage) -> void:
	_log.chat("Client %s" % senderid, packet.get_msg())

func _on_line_edit_text_submitted(text: String) -> void:
	var packet = packets.Packet.new()
	var chat_msg = packet.new_chat()
	chat_msg.set_msg(text)
	
	var err = WS.send(packet)
	if err != OK:
		_log.error("Error sending chat: %s" % err)
	else:
		_log.chat("You", text)
	_line_edit.clear()

	
