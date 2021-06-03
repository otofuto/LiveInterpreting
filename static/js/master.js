if ((serviceTitle = document.getElementsByClassName("service-title")).length > 0) {
	serviceTitle[0].addEventListener("click", () => {
		location = "/home/";
	});
}

document.documentElement.style.setProperty('--vh', window.innerHeight + 'px');
window.onresize = () => {
	document.documentElement.style.setProperty('--vh', window.innerHeight + 'px');
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

function logout() {
	fetch('/Logout/', {
		method: "post",
		credentials: "include"
	}).then(res => {
		if (res.status == 200)
			return {};
		else
			return null;
	}).then(result => {
		if (result != null) {
			location = "/";
		} else {
			alert('エラーによりログアウトに失敗しました。');
		}
	});
}

function post(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'post',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.text();
			return 'null';
		}).then(text => {
			try {
				let result = JSON.parse(text);
				if (result != null) {
					resolve(result);
				} else {
					reject(null);
				}
			} catch (err) {
				console.error(err);
				console.log(text);
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function get(url) {
	return new Promise((resolve, reject) => {
		fetch(url)
		.then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function put(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'put',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function del(url, data) {
	return new Promise((resolve, reject) => {
		fetch(url, {
			method: 'delete',
			body: data == null ? new FormData() : data,
			credentials: "include"
		}).then(res => {
			if (res.status == 200)
				return res.json();
			return null;
		}).then(result => {
			if (result != null) {
				resolve(result);
			} else {
				reject(null);
			}
		}).catch(err => {
			console.error(err);
			reject(err);
		});
	});
}

function formDisabled(form, dis) {
	if (dis) {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.setAttribute('disabled', ''));
	} else {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.removeAttribute('disabled'));
	}
}