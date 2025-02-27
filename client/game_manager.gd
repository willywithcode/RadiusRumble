extends Node
enum GameState {
	ENTERED,
	INGAME,
	CONNECTED,
}


var _state_scene: Dictionary = {
	GameState.ENTERED: "res://states/entered/entered.tscn",
	GameState.INGAME: "res://states/ingame/ingame.tscn",
	GameState.CONNECTED: "res://states/connected/connected.tscn",
}


var client_id: int
var _current_scene_root: Node

func _set_state(new_state: GameState) -> void:
	if _current_scene_root != null:
		_current_scene_root.queue_free()

	var scene: PackedScene = load(_state_scene[new_state])
	_current_scene_root = scene.instantiate()

	add_child(_current_scene_root)
