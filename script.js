$(document).ready(function() {

	$('.topics-form').submit(function(event) {
		event.preventDefault();

		var topicName = $('#topic').val();
		var card = $('.card');
		var input = $('.input');

		input.blur();
		card.addClass('saving');

		$('.subscribed-topic-name').text(topicName);
		$('.topic-messages').fadeIn('slow');

		if (typeof(EventSource) !== "undefined") {
			var source = new EventSource("http://localhost:8080/infocenter/" + topicName);

			source.onmessage = function(event) {
				var data = JSON.parse(event.data);

				$('.messages').append('<p class="message"><span class="message-id">' + data.id + ':</span> ' + data.msg + '</p>')
			};
		} else {
			$('.messages').html("Sorry, your browser does not support server-sent events...");
		}
	});

	$('.messages-form').submit(function(event) {
		event.preventDefault();

		var message = $('#message').val();
		var topicName = $('#topic').val();

		$('#message').val('');

        var xhr = new XMLHttpRequest();
        xhr.open('POST', 'http://localhost:8080/infocenter/'+topicName, true);
        xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');

        var data = {"msg": message};

        xhr.send(JSON.stringify(data));
	});
});