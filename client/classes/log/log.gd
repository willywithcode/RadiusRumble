class_name Log extends RichTextLabel

func _message(message: String, color: Color = Color.WHITE) -> void:
	append_text("[color=#%s]%s[/color]\n" % [color.to_html(), message])

func info(message: String) -> void:
	_message(message, Color.WHITE)

func warning(message: String) -> void:
	_message(message, Color.YELLOW)

func error(message: String) -> void:
	_message(message, Color.RED)

func success(message: String) -> void:
	_message(message, Color.GREEN)

func chat(sender: String, message: String) -> void:
	_message("[color=#%s]%s[/color]: %s" % [Color.WHITE.to_html(), sender, message])

