
# Around

This project is a platform for the user to share their exciting event which based on Geo-index. Users on this platform can post event they interested with pictures. Users also can search the nearby event by photo gallery mode or Map mode.

## Demo

This Demo will show you the basic signup login post and search by different mode. 
Since the Google Developer Tools are expensive so I only provide the Demo there.
![restaurant demo](https://user-images.githubusercontent.com/16642141/46192637-becb2700-c2c9-11e8-8551-db73916908b5.gif)

### Backend Description: 
#### Golang
This project's backend is built by Golang which is a most popular backend language. Golang has a great balance between development and work efficiency. Its Goroutines has a fantastic performance on concurrency make Go is a language easy to be scalable. 
For my project, even though I did not implement the Goroutines, I am familiar with the basic go script and handling mechanism. I create a main.go which process two main requests from the client, POST and GET.

### DataBase
ElasticSearch:   
MongoDB + Geo Indexing + Query Optimization  -- The main storage I used

 Pros：    
 - As a search engine it provides a powerful geo-index searching.   
 - As the necessary component for my project, it also offers limited memory to store the data which support the individual developer to test.   
 - It is free, and I need to download and set up the environment on Google Cloud Engine. 
 
 Cons:  
 - It is not a permanent database. Once this project will be used as the business, it will lose the data if we still want to use it as the primary database.   
   
Bigtable:  
Cloud + MongoDB (online)   -- Database as mainstream  
Pros:  
 - The permanent database provides stable data storage.  
 - NoSQL database with great scalability   

Cons:  

 - Much expensive for the starter.   
 - Not complex data query since it is a NoSQL.   

BigQuery:  
Cloud + MySQL (offline analysis)  
Pro:  

 - Powerful query method for analysis. 
  
Con:  
 - off line
 - It needs Dataflow to dump all data from BigTable.   

Database tools:  

 - GCS: For unstructured data, in this project, we used it as image storage. 
 - Dataflow: Dump data from BigTable for BigQuery.  
 - Redis: As a cache, optimize search processing time.   


End with an example of getting some data out of the system or using it for a little demo

## Front End

Implemented by ReactJS. Main libraries I used are from  **antd, react-google-maps and react-grid-gallery**.


## Built With

* [ReactJS](https://reactjs.org/) - The web framework used
* [Go](https://golang.org/) - Backend framework and language 
* [Antd](https://github.com/ant-design/ant-design) - Most useful front end Library

## Authors

* **Bruce Guo** - *Initial work* - [Around](https://github.com/Brucegx/Around)

See also the list of [contributors](https://github.com/Brucegx/Around/contributors) who participated in this project.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
