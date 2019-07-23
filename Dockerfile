FROM node:10.9.0

# Create app directory
WORKDIR /Users/src/twilio-communication-app

# Install app dependencies for backend code
COPY package.json ./
RUN yarn
RUN rm -rf node_modules
RUN yarn cache clean
RUN yarn install

# Bundle backend source
COPY . .

EXPOSE 3000
CMD [ "yarn", "start" ]
