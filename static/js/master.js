if ((serviceTitle = document.getElementsByClassName("service-title")).length > 0) {
	serviceTitle[0].addEventListener("click", () => {
		location = "/";
	});
}

function checkLogin() {
	return new Promise((resolve, reject) => {
		fetch('/Login/', {
			method: "get",
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.json();
			else
				return null;
		})
		.then(result => {
			resolve(result);
		});
	});
}