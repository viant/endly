
$(document).ready(function () {

    $.ajax({
        dataType: "json",
        url: '/v1/api/dummy',
        success: function (resp) {
            var data = resp.Data
            console.log(data);
            for(var i = 0; i<data.length;i++) {
                var item = data[i];
                var typeName = '';
                if(item.Type) {
                    typeName = item.Type.Name
                }
                $('#table').append('<tr><td/><td>' + item.Id + '</td><td>'+ item.Name+ '</td><td>'+ typeName+' </td></tr>');
            }
        }
    });


});
