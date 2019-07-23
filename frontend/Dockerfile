FROM node:10.9.0

# Create app directory
WORKDIR /Users/src/twilio-communication-app/front-end

## Install app dependencies for frontend code
COPY package.json ./
RUN rm -rf node_modules
RUN yarn cache clean
RUN yarn install
COPY . .

EXPOSE 4200
CMD [ "yarn", "build" ]
CMD [ "yarn", "start" ]
