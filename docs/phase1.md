The first part of this project was setting up docker with wsl2 ubuntu integration, and then learning how to set up my containerized services.

The first service was from a public image (Pihole), it was set up just by grabbing the compose file from the pihole docker github repo. 
Then i made a .env file with the admin endpoint password and then i was able to set up my first container. 

For the second service i made my own service. A python app using Flask and some other time imports. Just a basic api with two endpoints one being just a welcome to my homelab and the other a time counter for how long the service has been running. Crucially the python code has this piece of code:
                    if __name__ == "__main__":
                        app.run(host="0.0.0.0", port=5000)
This code makes the app listen to requests from all network interfaces.

Then i made a dockerfile for the app that installs Flask from requirements.txt and all that other good stuff like choosing the python image 3.12-slim and making the working directory. 

Then i made the docker-compose for my image which just builds from the dockerfile in the dir, gives a name to the container, and makes the tunnel/opens the ports between windows and the container. And makes the container auto restart unless stopped manually. 

The way i have it setup right now is each servicefolder has its own docker-compose, and then i have a more orchestrating compose at the root. 

Some mistakes ive made so far:
i had the env file in pihole directory, but if we are gonna use the orchestration compose at root we need the env at root.

Obviously i made tons of mistakes making the python app but thats a given since i suck at python. 

Also have had lots of struggles with wsl2 integration with docker and stuff.

