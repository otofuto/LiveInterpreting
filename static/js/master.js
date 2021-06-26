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
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
	} else {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.removeAttribute('disabled'));
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.removeAttribute('onclick'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.removeAttribute('onclick'));
	}
}

function userName(uid) {
	return new Promise((resolve, reject) => {
		get('/Account/name/' + uid)
		.then(res => {
			resolve(res.name);
		}).catch(err => {
			reject(err);
		});
	});
}

function getNotifTypeMessage(ntype) {
	let ret = "";
	switch (ntype) {
		case 'dm':
			ret = "ダイレクトメッセージが届いています";
			break;
		case 'trans/req':
			ret = "見積依頼が届いています";
			break;
		case 'trans/res':
			ret = "見積が届いています";
			break;
		case 'trans/buy':
			ret = '見積が購入されました';
			break;
		default:
			ret = "不明なメッセージ";
			break;
	}
	return ret;
}

function formatdate(str, timeView = true) {
	let dt = new Date(str);
	let ret = dt.getFullYear() + '年 ' +
		(dt.getMonth() + 1) + '月 ' + dt.getDate() + '日';
	if (timeView) ret += ' ' + frontZero(dt.getHours()) + ':' + frontZero(dt.getMinutes());
	return ret;
}

function frontZero(s) {
	if ((s - 0) < 10) s = '0' + s;
	return s;
}