import { Buffer } from 'node:buffer'

export async function pre_get(endpoint) {
    const getRandomToken = await fetch("http://127.0.0.1:1234/token")
    const randomTokenWithStuff = await getRandomToken.text()
    const nextFetch = await fetch(`http://127.0.0.1:1235/${endpoint}`, {
        headers: {
            "sideline": randomTokenWithStuff
        }
    })
    return await nextFetch.text()
}

export async function register() { // this throws a menu in the same url and the background gets shadowed and darker except the menu itself and the menu has the masterImage and titleImage and user should give the answer with the submit button
    const ctrf_token = await pre_get("register")
    const registerFetch = await fetch("http://127.0.0.1:1235/register", {
        headers: {
            "ctrf-token": ctrf_token
        },
        method: "POST"
    })
    const registerJson = await registerFetch.json()
    const masterImage = atob(registerJson.masterImage)
    const titleImage = atob(registerJson.titleImage)
    return { titleImage: titleImage, masterImage: masterImage, status: registerFetch.status }
}

export async function registerValidation(email, x, y) { // this is the validation process of the answer and if it was true, important thing to know is we are still in /register url
    const ctrf_token = await pre_get("register/validation")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation", {
        headers: {
            "ctrf-token": ctrf_token,
            "captchaAnswer": `{"x":"${x}","y":"${y}"}`,
            "email": email
        },
        method: "POST"
    })
    return registerFetch.status
}

export async function registerValidationJWT(username, password) { // now we got redirected from /register to /register/validation/jwt
    const ctrf_token = await pre_get("register/validation/jwt")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/jwt", {
        headers: {
            "ctrf-token": ctrf_token,
            "username": username,
            "password": password
        },
        method: "POST"
    })
    return registerFetch.status
}

export async function registerValidationSubmit(vcode) { // we are still in /register/validation/jwt
    const ctrf_token = await pre_get("register/validation/submit")
    const registerFetch = await fetch("http://127.0.0.1:1235/register/validation/submit", {
        headers: {
            "ctrf-token": ctrf_token,
            "verification": vcode,
        },
        method: "POST"
    })

    return registerFetch.status
}