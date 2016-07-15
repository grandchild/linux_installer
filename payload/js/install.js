

window.onbeforeunload = function() {
	alert("YO!!");
	$.get('/quit');
};
