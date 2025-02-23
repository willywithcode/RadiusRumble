extends Node

const packets := preload("res://packets.gd")

func _ready() -> void:
    var packet = packets.Packet.new()
    packet.set_sender_id(1)
    var chat_msg = packet.new_chat()
    chat_msg.set_msg("Hello, World!")
    var data = packet.to_bytes()
    print(data)
