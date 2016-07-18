

var os = '';
var config = {};
var locale = '';
var strings = {};

$(document).ready(function() {
	pushHandler();
	$.get('/os', function(data) { os = data.toLowerCase();
	$.getJSON('/install.config', function(json) { config = json;
	$.get('/locale', function(data) { locale = data;
	$.getJSON('/strings', function(json) { strings = json;
	loadInternationalStrings();
	loadLanguage(locale);
	})})})});
});

function loadInternationalStrings() {
	for(var lang in strings) {
		if(strings.hasOwnProperty(lang)) {
			$('.ui-string.choose_language.'+lang).html(strings[lang]['choose_language']);
		}
	}
}

function loadLanguage(lang) {
	for(var key in strings[lang]) {
		if(strings[lang].hasOwnProperty(key)) {
			var string = expandStringVariables(strings[lang][key], strings[lang]);
			if (key==='title') {
				document.title = string;
			} else {
				$('.ui-string.'+key).html(string);
			}
		}
	}
}

function expandStringVariables(string, data) {
	string = string.replace(/\$PRODUCT/, config['$PRODUCT']);
	string = string.replace(/\$VERSION/, config['$VERSION']);
	string = string.replace(/\$APPLAUNCHER/, data['_'+os+'_app_launcher']);
	return string;
}

function pushHandler() {
	$.get('/push', function(data) {
		switch(data) {
		case 'quit':
			close();
			alert('Installer was quit outside the browser. You should close this tab.')
			break;
		case 'refresh push':
			pushHandler();
			break;
		default:
			console.error('Unknown command from push channel: \''+data+'\'')
		}
	})
}

window.onbeforeunload = function(e) {
	// $.get('/quit');
};
