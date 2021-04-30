#!/usr/bin/env node

const _ = require('lodash');
const yargs = require('yargs');
const axios = require('axios');
const productApi = 'https://api.tangiblee.com/api/tngimpr?ids=%s&domain=www.coachoutlet.com&activeLocale=en-US';
const productPage = 'https://www.coachoutlet.com/products/%s1.html?dwvar_color=%s2'
const interval = 15 * 1000;

// twilio credentials
const config = require('../local-credentials.json');
const { string } = require('yargs');
const accountSid = config.accountSid;
const authToken = config.authToken;
const twilioClient = require('twilio')(accountSid, authToken);

const options = yargs
  .usage("Usage: -i <product-id>")
  .option("i", { alias: "id", describe: "Desired product id", type: "string", demandOption: true })
  .argv;

setInterval(() => {
  axios.get(_.replace(productApi, '%s', options.id))
    .then((res) => {
      const productId = options.id.toLowerCase();
      if (_.get(res, `data.exists.${productId}`)) {
        console.log(`product ${productId} exists!`);
        sendMessage(productId);
      } else {
        console.log(`product ${productId} doesn't exist!`);
      }
    })
    .catch((err) => {
      console.error('got the error: ', err);
    })
}, interval + Math.floor(Math.random() * 5 * 1000));

function sendMessage(productId) {
  const productPageById = getProductPageById(productId);
  const textMessage = `Your coach product ${productId} has stocks left. Check the link: ${productPageById}`;
  console.log("text message is " + textMessage);
  twilioClient.messages
    .create({
      body: textMessage,
      from: config.fromNumber,
      to: config.toNumber,
    })
    .then(message => console.log(message.sid));
}

function getProductPageById(productId) {
  const productElements = _.split(productId, '_');
  const productNumber = _.get(productElements, '[0]');
  const productColor = _.get(productElements, '[1]');
  const tempProductPage = _.replace(productPage, '%s1', productNumber.toUpperCase())

  return _.replace(tempProductPage, '%s2', productColor.toUpperCase())
}